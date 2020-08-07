package vod

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/h2non/filetype"
)

// VideoResizeCommand resizes the video provided and writes the new file to the filesystem
// TODO: send output to write stream
func VideoResizeCommand(cmd *exec.Cmd, input io.Reader, output io.Writer) error {
	cmd.Stdin = input
	cmd.Stdout = output
	err := cmd.Run()
	if err != nil {
		log.Printf("%s", err.Error())
		return errors.New("Failed to start video resize process")
	}

	return nil
}

// ThumbnailCommand generates thumbnail from video input and sets it to the output reader.
func ThumbnailCommand(cmd *exec.Cmd, input io.Reader, output io.Writer) error {
	cmd.Stdin = input
	cmd.Stdout = output
	err := cmd.Run()
	if err != nil {
		log.Printf("%s", err.Error())
		return errors.New("Failed to start thumbnail process")
	}

	return nil
}

// GetDimension returns the dimension of video from stream
func GetDimension(video io.Reader) (*Dimension, error) {
	cmd := exec.Command("ffprobe",
		"-i", "pipe:0",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0",
	)

	out := new(strings.Builder)
	cmd.Stdin = video
	cmd.Stdout = out
	err := cmd.Run()
	if err != nil {
		log.Printf(err.Error())
		return nil, errors.New("Error occurred while extracting dimension")
	}

	outString := out.String()
	data := strings.Split(strings.Trim(outString, " \n"), "x")
	x, _ := strconv.Atoi(data[0])
	y, _ := strconv.Atoi(data[1])
	result := Dimension{width: x, height: y}

	if result.height <= 10 || result.width <= 10 {
		return nil, errors.New("Input file has invalid resolution")
	}

	return &result, nil
}

// ProcessVideoInput processes the video input
func ProcessVideoInput(input *os.File, contentType string) error {
	// Before processing file, move reader to begining to avoid errors
	input.Seek(0, 0)

	// Get the current video dimension in order to calculate resizing
	dimen, err := GetDimension(input)
	if err != nil {
		return err
	}

	destinationRoot := getFilePath()
	var output1080 bytes.Buffer
	var output720 bytes.Buffer
	var outputThumb bytes.Buffer

	// Current design is to have 1080p and 720p resolutions available
	// First check video size before resizing
	if dimen.height > 1080 {
		// Video is larger than 1080, hence proceed to resizing step

		// After dimension read, move file reader to beginning once more
		input.Seek(0, 0)
		err = startVideoProcess(input, &output1080, VideoSizes["1080p"])
		if err != nil {
			log.Println("File processing failed!")
			return nil
		}
		err = completeRequest(&output1080, contentType, destinationRoot+"/1080")
		if err != nil {
			log.Println("File processing failed for 1080 video!")
			return err
		}
	}

	// Check if the file is larger than 720p
	if dimen.height > 720 {
		// Move file reader to beginning once more... Not sure what got executed last... Just a precaution
		input.Seek(0, 0)
		err = startVideoProcess(input, &output720, VideoSizes["720p"])
		if err != nil {
			log.Println("File processing failed!")
			return err
		}
		err = completeRequest(&output720, contentType, destinationRoot+"/720")
		if err != nil {
			log.Println("File processing failed for 720 video!")
			return err
		}
	}

	input.Seek(0, 0)
	err = generateThumbnail(input, &outputThumb, "00:00:03")
	if err != nil {
		return err
	}
	err = completeRequest(&outputThumb, http.DetectContentType(outputThumb.Bytes()), destinationRoot+"/thumb")
	if err != nil {
		log.Println("File processing failed for image!")
		return err
	}

	return nil
}

func generateThumbnail(input *os.File, outputThumb io.Writer, time string) error {
	cmd := exec.Command("ffmpeg",
		"-ss", time, "-i", "pipe:0",
		"-frames:v", "1",
		"-f", "image2",
		"-vf", fmt.Sprintf("scale=%s:%s:force_original_aspect_ratio=decrease", strconv.Itoa(600), strconv.Itoa(600)),
		"pipe:1",
	)

	err := ThumbnailCommand(cmd, input, outputThumb)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

// CreateVideoServer is used to process upload post request.
// The desired workflow is to get the initial video data into a file and feed that file to the ffmpeg process.
func CreateVideoServer(r *gin.Engine, config *Configuration) *gin.RouterGroup {
	g := r.Group("/video")
	g.POST("/file", func(c *gin.Context) {
		//c.Request.ParseMultipartForm(config.MaxUploadSize)
		rawFile, _, err := c.Request.FormFile("upload")
		if err != nil {
			log.Printf(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read uploaded data"})
			return
		}
		file, ok := rawFile.(*os.File)
		if !ok {
			log.Printf(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read uploaded data"})
			return
		}
		defer file.Close()
		defer os.Remove(file.Name())

		buf := bufio.NewReaderSize(file, 600)
		head, err := buf.Peek(512)
		isVideoType, contentType := IsVideo(head)
		if err != nil || (!filetype.IsVideo(head) && !isVideoType) {
			if err != nil {
				log.Printf(err.Error())
			} else {
				log.Printf("Not a video file")
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate stream"})
			return
		}

		err = ProcessVideoInput(file, contentType)
		if err != nil {
			log.Printf(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot proceed with processing due to internal error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Successfully processed data"})

	})

	g.POST("/stream", func(c *gin.Context) {
		// Get uploaded file
		reader := c.Request.Body
		buf := bufio.NewReaderSize(reader, 600)
		head, err := buf.Peek(512)
		isVideoType, contentType := IsVideo(head)
		if err != nil || (!filetype.IsVideo(head) && !isVideoType) {
			if err != nil {
				log.Printf(err.Error())
			} else {
				log.Printf("Not a video file")
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate stream"})
			return
		}
		// Save incoming file
		newFile, err := ioutil.TempFile("", "upload-*")
		if err != nil {
			log.Printf(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot proceed with processing due to internal error"})
			return
		}
		defer newFile.Close()
		defer os.Remove(newFile.Name())

		// Write the current data to filesystem
		_, err = buf.WriteTo(newFile)
		defer reader.Close()

		if err != nil {
			log.Printf(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot proceed with processing due to internal error"})
			return
		}
		err = ProcessVideoInput(newFile, contentType)
		if err != nil {
			log.Printf(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot proceed with processing due to internal error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Successfully processed data"})
	})

	g.POST("/resize", func(c *gin.Context) {
		rawURL, exists := c.GetQuery("url")
		if !exists {

		}
		sourceURL, err := url.QueryUnescape(rawURL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Url must be provided"})
			return
		}
		// Save incoming file
		newFile, err := ioutil.TempFile("", "upload-*")
		if err != nil {
			log.Printf(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot proceed with processing due to internal error"})
			return
		}
		defer newFile.Close()
		defer os.Remove(newFile.Name())

		err = downloadData(sourceURL, newFile)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse Url"})
			return
		}

		newFile.Seek(0, 0)
		buf := bufio.NewReaderSize(newFile, 600)
		head, err := buf.Peek(512)
		isVideoType, contentType := IsVideo(head)
		if err != nil || (!filetype.IsVideo(head) && !isVideoType) {
			if err != nil {
				log.Printf(err.Error())
			} else {
				log.Printf("Not a video file")
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate stream"})
			return
		}
		err = ProcessVideoInput(newFile, contentType)
		if err != nil {
			log.Printf(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot proceed with processing due to internal error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Successfully processed data"})
	})
	return g
}

func startVideoProcess(input io.Reader, outputVideo io.Writer, d Dimension) error {
	cmd := exec.Command("ffmpeg",
		"-i", "pipe:0",
		"-movflags", "frag_keyframe+empty_moov", "-f", "mp4",
		"-vf", fmt.Sprintf("scale=%s:%s:force_original_aspect_ratio=decrease", strconv.Itoa(d.width), strconv.Itoa(d.height)),
		"pipe:1",
	)

	err := VideoResizeCommand(cmd, input, outputVideo)
	if err != nil {
		log.Println("File processing failed!")
		return err
	}

	return nil
}
