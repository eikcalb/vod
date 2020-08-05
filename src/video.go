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
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/h2non/filetype"
)

const INVALID_SIZE = iota

// VideoResizeCommand resizes the video provided and writes the new file to the filesystem
// TODO: send output to write stream
func VideoResizeCommand(cmd *exec.Cmd, input io.ReadCloser, output io.Writer) (int64, error) {
	// stdin for process invoked
	pipe, err := cmd.StdinPipe()
	if err != nil {
		log.Panic(err)
		return INVALID_SIZE, errors.New("Failed to read data from stream for video processing")
	}

	// stdout for process invoked
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Panic(err)
		return INVALID_SIZE, errors.New("Failed to write data from stream for video processing")
	}

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
		return INVALID_SIZE, errors.New("Failed to start video resize process")
	}

	defer pipe.Close()
	defer stdout.Close()

	// Copy data from input to the running process
	_, err = io.Copy(pipe, input)

	if err != nil {
		log.Fatal(err)
		return INVALID_SIZE, errors.New("Failed to copy file to stream")
	}

	// Copy data from process output
	size, err := io.Copy(output, stdout)

	if err != nil {
		log.Fatal(err)
		return INVALID_SIZE, errors.New("Failed to copy file from stream")
	}

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
		return INVALID_SIZE, errors.New("Error occurred while running process")
	}

	return size, nil
}

// ThumbnailCommand generates thumbnail from video input and sets it to the output reader.
func ThumbnailCommand(cmd *exec.Cmd, input io.ReadCloser, output io.Writer) (int64, error) {
	pipe, err := cmd.StdinPipe()
	if err != nil {
		log.Panic(err)
		return INVALID_SIZE, errors.New("Failed to read data from stream for thumbnail generation")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Panic(err)
		return INVALID_SIZE, errors.New("Failed to write data from stream for thumbnail generation")
	}

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
		return INVALID_SIZE, errors.New("Failed to start thumbnail process")
	}

	defer pipe.Close()
	defer stdout.Close()

	_, err = io.Copy(pipe, input)

	if err != nil {
		log.Fatal(err)
		return INVALID_SIZE, errors.New("Failed to copy file to stream")
	}
	size, err := io.Copy(output, stdout)

	if err != nil {
		log.Fatal(err)
		return INVALID_SIZE, errors.New("Failed to copy file from stream")
	}

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
		return INVALID_SIZE, errors.New("Error occurred while running process")
	}

	return size, nil
}

// GetDimension returns the dimension of video from stream
func GetDimension(video *os.File) (*Dimension, error) {
	cmd := exec.Command("ffprobe",
		"-i", "pipe:0",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=height,width",
		"-of", "csv=s=x:p=0",
	)

	pipe, err := cmd.StdinPipe()
	if err != nil {
		log.Panic(err)
		return nil, errors.New("Failed to read data from stream for thumbnail generation")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Panic(err)
		return nil, errors.New("Failed to write data from stream for thumbnail generation")
	}

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
		return nil, errors.New("Failed to start thumbnail process")
	}

	defer pipe.Close()
	defer stdout.Close()

	_, err = io.Copy(pipe, video)

	if err != nil {
		log.Fatal(err)
		return nil, errors.New("Failed to copy file to stream")
	}
	var out strings.Builder
	_, err = io.Copy(&out, stdout)
	if err != nil {
		log.Fatal(err)
		return nil, errors.New("Failed to copy file from stream")
	}

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
		return nil, errors.New("Error occurred while running process")
	}

	outString := out.String()
	log.Printf("ffprobe output: %s", outString)
	data := strings.Split(outString, "x")
	x, _ := strconv.Atoi(data[0])
	y, _ := strconv.Atoi(data[1])
	result := Dimension{x, y}
	return &result, nil
}

func processVideoInput(input *os.File) error {
	d, err := GetDimension(input)
	if err != nil {
		return err
	}

	cmd := exec.Command("ffmpeg",
		"-i", "pipe:0",
		"-format", "mp4",
		"-vf", fmt.Sprintf("scale=%s:%s:force_original_aspect_ratio:decrease", strconv.Itoa(d.width), strconv.Itoa(d.height)),
		"pipe:1",
	)

	// This will create a new video and the output can be utilized for any storage medium.
	// This should be done within a loop
	var outputVideo bytes.Buffer
	VideoResizeCommand(cmd, input, &outputVideo)
	return err
}

func processThumbnail(input *os.File, d Dimension, time string) {
	cmd := exec.Command("ffmpeg",
		"-ss", time, "-i", "pipe:0",
		"-frames:v", "1",
		"-o", "pipe:1",
		"-format", "png",
		"-s", fmt.Sprintf("%sx%s", strconv.Itoa(d.width), strconv.Itoa(d.height)),
		"pipe:1",
	)
	var outputThumb bytes.Buffer
	ThumbnailCommand(cmd, input, &outputThumb)
}

// CreateVideoServer is used to process upload post request.
// The desired workflow is to get the initial video data into a file and feed that file to the ffmpeg process.
func CreateVideoServer(r *gin.Engine, config *Configuration) *gin.RouterGroup {
	g := r.Group("/video")
	g.POST("/file", func(c *gin.Context) {
		c.Request.ParseMultipartForm(config.MaxUploadSize)
		rawFile, header, err := c.Request.FormFile("upload")
		if err != nil {
			log.Fatal(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read uploaded data"})
			return
		}
		file, ok := rawFile.(*os.File)
		if !ok {
			log.Fatal(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read uploaded data"})
			return
		}

		log.Printf("new video file %s", header.Filename)
		processVideoInput(file)

	})

	g.POST("/stream", func(c *gin.Context) {
		// Get uploaded file
		reader, err := c.Request.GetBody()
		if err != nil {
			log.Fatal(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read uploaded data"})
			return
		}
		buf := bufio.NewReaderSize(reader, 300)
		head, err := buf.Peek(280)
		if err != nil || !filetype.IsVideo(head) {
			log.Fatal(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate stream"})
			return
		}
		newFile, err := ioutil.TempFile("", "upload-*")
		defer newFile.Close()
		defer reader.Close()

		if err != nil {
			log.Fatal(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot proceed with processing due to internal error"})
			return
		}
		// Write the current data to filesystem
		buf.WriteTo(newFile)

		processVideoInput(newFile)
	})

	return g
}