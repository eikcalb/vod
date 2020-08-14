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

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"

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
		log.Printf("Failed to start video resize process")
		return err
	}

	return nil
}

// ThumbnailCommand generates thumbnail from video input and sets it to the output reader.
func ThumbnailCommand(cmd *exec.Cmd, input io.Reader, output io.Writer) error {
	cmd.Stdin = input
	cmd.Stdout = output
	err := cmd.Run()
	if err != nil {
		log.Printf("Failed to start thumbnail process")
		return err
	}

	return nil
}

// GetDimension returns the dimension of video from stream
func GetDimension(video io.Reader) (*Dimension, error) {
	cmd := exec.Command("ffprobe",
		"-t", "3s",
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

// GetDuration returns the duration of video from stream
func GetDuration(video io.Reader) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-i", "pipe:0",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
	)

	out := new(strings.Builder)
	cmd.Stdin = video
	cmd.Stdout = out
	err := cmd.Run()
	if err != nil {
		log.Printf(err.Error())
		return 0, err
	}
	outString := out.String()
	durationSeconds, err := strconv.ParseFloat(strings.Trim(outString, " \n"), 32)
	if err != nil {
		return 0, err
	}
	if durationSeconds < 0 {
		return 0, errors.New("Input file has invalid duration")
	}

	return durationSeconds, nil
}

// ProcessVideoInput processes the video input.
// The assumption is that all videos received are 1080p.
// It is only required to resize once to 720p
func ProcessVideoInput(input *os.File, contentType string) error {
	// Before processing file, move reader to begining to avoid errors
	input.Seek(0, 0)

	destinationRoot := generatePath("media/")
	var output1080 bytes.Buffer
	var output720 bytes.Buffer
	var outputThumb bytes.Buffer

	err := completeRequest(&output1080, contentType, destinationRoot+"/1080.mp4")
	if err != nil {
		log.Println("File processing failed for 1080 video!")
		return err
	}

	// Generate 720 video
	err = startVideoProcess(input, &output720, VideoSizes["720p"])
	if err != nil {
		log.Println("File processing failed!")
		return err
	}
	err = completeRequest(&output720, contentType, destinationRoot+"/720.mp4")
	if err != nil {
		log.Println("File processing failed for 720 video!")
		return err
	}

	input.Seek(0, 0)
	// Generate thumbnail
	err = generateThumbnail(input, &outputThumb, "00:00:03")
	if err != nil {
		return err
	}
	err = completeRequest(&outputThumb, http.DetectContentType(outputThumb.Bytes()), destinationRoot+"/thumb.png")
	if err != nil {
		log.Println("File processing failed for image!")
		return err
	}

	return nil
}

func generateThumbnail(input io.Reader, outputThumb io.Writer, time string) error {
	cmd := exec.Command("ffmpeg",
		"-ss", time, "-i", "pipe:0",
		"-frames:v", "1",
		"-f", "image2",
		"-vf", fmt.Sprintf("scale=%s:%s:force_original_aspect_ratio=decrease,pad=%s:%s:(ow-iw)/2:(oh-ih)/2", "600", "600", "600", "600"),
		"pipe:1",
	)

	err := ThumbnailCommand(cmd, input, outputThumb)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func generateThumbnailWithFile(input os.File, outputThumb io.Writer, time string, size Dimension) error {
	cmd := exec.Command("ffmpeg",
		"-ss", time, "-i", input.Name(),
		"-frames:v", "1",
		"-f", "image2",
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2", size.width, size.height, size.width, size.height),
		"pipe:1",
	)
	cmd.Stdout = outputThumb
	err := cmd.Run()
	if err != nil {
		log.Printf("Failed to start thumbnail process")
		return err
	}

	return nil
}

// CreateVideoServer is used to process upload post request.
// The desired workflow is to get the initial video data into a file and feed that file to the ffmpeg process.
func CreateVideoServer(r *gin.Engine, config *Configuration) *gin.RouterGroup {
	g := r.Group("/findapp")
	g.POST("/gemform", func(c *gin.Context) {
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

	g.POST("/gem", func(c *gin.Context) {
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

	g.PATCH("/gem", func(c *gin.Context) {
		rawFileKey, exists := c.GetQuery("url")
		if !exists {

		}
		sourceKey, err := url.QueryUnescape(rawFileKey)
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

		err = downloadData(sourceKey, newFile, Config.AWS.InputBucketName)
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

// HandleAWSMediaOld is called in lambda upon activity in a lambda
func HandleAWSMediaOld(s3 events.S3Entity) error {
	inputData := aws.NewWriteAtBuffer([]byte{})
	fileKey, err := url.QueryUnescape(s3.Object.Key)
	if err != nil {
		return err
	}

	err = downloadData(fileKey, inputData, Config.AWS.InputBucketName)
	if err != nil {
		return err
	}

	bytesRead := inputData.Bytes()
	reader := bytes.NewReader(bytesRead[:])
	head := bytesRead[:512]
	isVideoType, contentType := IsVideo(head)
	if err != nil || (!filetype.IsVideo(head) && !isVideoType) {
		if err != nil {
			return err
		}
		return errors.New("Cannot proceed with processing due to internal error")
	}
	destinationRoot := getMediaFilePath(fileKey)

	// Copy root file to output bucket
	err = copyData(fileKey, destinationRoot+"/1080.mp4", contentType)
	if err != nil {
		return err
	}

	var output720 bytes.Buffer
	var outputThumb bytes.Buffer
	err = startVideoProcess(reader, &output720, VideoSizes["720p"])
	if err != nil {
		return err
	}
	err = completeRequest(&output720, contentType, destinationRoot+"/720.mp4")
	if err != nil {
		return err
	}

	reader.Seek(0, 0)
	err = generateThumbnail(reader, &outputThumb, "00:00:03")
	if err != nil {
		return err
	}
	err = completeRequest(&outputThumb, http.DetectContentType(outputThumb.Bytes()), destinationRoot+"/thumb.png")
	if err != nil {
		return err
	}

	return nil
}

// HandleAWSMedia is called in lambda upon activity in a lambda
func HandleAWSMedia(s3 events.S3Entity) error {
	inputData := aws.NewWriteAtBuffer([]byte{})
	fileKey, err := url.QueryUnescape(s3.Object.Key)
	if err != nil {
		return err
	}

	// Download the uploaded data from S3
	err = downloadData(fileKey, inputData, Config.AWS.InputBucketName)
	if err != nil {
		return err
	}

	// Create a temp file which will be used for storing the video.
	// This is because tests proved that ffmpeg processes files better than streams.
	tempFile, err := ioutil.TempFile("", "upload-*.mp4")
	if err != nil {
		return err
	}
	bytesRead := inputData.Bytes()
	err = ioutil.WriteFile(tempFile.Name(), bytesRead, 0666)
	if err != nil {
		return err
	}

	// Test if input is actually a video file
	head := bytesRead[:512]
	isVideoType, contentType := IsVideo(head)
	if err != nil || (!filetype.IsVideo(head) && !isVideoType) {
		if err != nil {
			return err
		}
		fmt.Printf("Cannot proceed with processing due to internal error, %v", err)
		// If file is not a video, do not return an error to prevent lambda from being rerun.
		return nil
	}
	reader := bytes.NewReader(bytesRead[:])
	rawDuration, err := GetDuration(reader)
	if err != nil {
		return err
	}
	duration := strconv.FormatFloat((rawDuration / 2), 'f', 4, 64)
	destinationRoot := getMediaFilePath(fileKey)

	// Copy root file to output bucket
	err = copyData(fileKey, destinationRoot+"/1080.mp4", contentType)
	if err != nil {
		return err
	}

	// For each output, create a buffer and save the converted data.
	var output720 bytes.Buffer
	var outputThumb bytes.Buffer

	// Generate resized video for 720p --- START
	err = startVideoProcessWithFile(*tempFile, &output720, VideoSizes["720p"])
	if err != nil {
		return err
	}
	err = completeRequest(&output720, contentType, destinationRoot+"/720.mp4")
	if err != nil {
		return err
	}
	// 720p --- END

	// Generate thumbnails 1080 --- START
	err = generateThumbnailWithFile(*tempFile, &outputThumb, duration, VideoSizes["1080p"])
	if err != nil {
		return err
	}
	imageType := http.DetectContentType(outputThumb.Bytes())
	err = completeRequest(&outputThumb, imageType, destinationRoot+"/1080.jpg")
	if err != nil {
		return err
	}
	// 1080 --- END

	// 720 --- START
	// Reuse buffer
	outputThumb.Reset()
	err = generateThumbnailWithFile(*tempFile, &outputThumb, duration, VideoSizes["720p"])
	if err != nil {
		return err
	}
	err = completeRequest(&outputThumb, imageType, destinationRoot+"/720.jpg")
	if err != nil {
		return err
	}
	// 720 --- END

	// 200 --- START
	// Reuse buffer
	outputThumb.Reset()
	err = generateThumbnailWithFile(*tempFile, &outputThumb, duration, Dimension{200, 200})
	if err != nil {
		return err
	}
	err = completeRequest(&outputThumb, imageType, destinationRoot+"/200.jpg")
	if err != nil {
		return err
	}
	// 200 --- END

	return nil
}

func startVideoProcess(input io.Reader, outputVideo io.Writer, d Dimension) error {
	width, height := strconv.Itoa(d.width), strconv.Itoa(d.height)
	cmd := exec.Command("ffmpeg",
		"-i", "pipe:0",
		"-movflags", "frag_keyframe+empty_moov", "-f", "mp4",
		"-vf", fmt.Sprintf("scale=%s:%s:force_original_aspect_ratio=decrease,pad=%s:%s:(ow-iw)/2:(oh-ih)/2", width, height, width, height),
		"pipe:1",
	)

	err := VideoResizeCommand(cmd, input, outputVideo)
	if err != nil {
		log.Println("File processing failed!")
		return err
	}

	return nil
}

func startVideoProcessWithFile(input os.File, outputVideo io.Writer, d Dimension) error {
	width, height := strconv.Itoa(d.width), strconv.Itoa(d.height)
	cmd := exec.Command("ffmpeg",
		"-i", input.Name(),
		"-movflags", "frag_keyframe+empty_moov", "-f", "mp4",
		"-vf", fmt.Sprintf("scale=%s:%s:force_original_aspect_ratio=decrease,pad=%s:%s:(ow-iw)/2:(oh-ih)/2", width, height, width, height),
		"pipe:1",
	)

	cmd.Stdout = outputVideo
	err := cmd.Run()
	if err != nil {
		log.Println("File processing failed!")
		return err
	}

	return nil
}
