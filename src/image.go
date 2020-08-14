package vod

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/gin-gonic/gin"
	"github.com/h2non/filetype"

	"github.com/aws/aws-sdk-go/aws"
)

// CreateImageServer creates an image server
func CreateImageServer(r *gin.Engine) *gin.RouterGroup {
	g := r.Group("/findapp")

	g.POST("/catalogue", func(c *gin.Context) {
		reader := c.Request.Body
		buf := bufio.NewReaderSize(reader, 600)
		head, err := buf.Peek(512)
		contentType := http.DetectContentType(head)
		defer reader.Close()
		if err != nil || (!strings.HasPrefix(contentType, "image") && !filetype.IsVideo(head)) {
			log.Printf(err.Error())
			c.JSON(http.StatusNotAcceptable, gin.H{"error": "Cannot proceed with processing due to internal error"})
			return
		}

		var out bytes.Buffer
		err = ResizeImage(buf, &out, *NewDimension(600, 600))
		if err != nil {
			log.Printf(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot proceed with processing due to internal error"})
			return
		}

		completeRequest(&out, contentType, generatePath("catalogue/")+"/600.png")
		c.JSON(http.StatusOK, gin.H{"message": "Successfully processed data"})
	})

	return g
}

// HandleAWSCatalogue is called in lambda upon activity in a lambda
func HandleAWSCatalogue(s3 events.S3Entity) error {
	inputData := aws.NewWriteAtBuffer([]byte{})
	fileKey, err := url.QueryUnescape(s3.Object.Key)
	if err != nil {
		return err
	}
	err = downloadData(fileKey, inputData, Config.AWS.InputBucketName)
	if err != nil {
		return err
	}
	imageBytes := inputData.Bytes()
	bytesReader := bytes.NewReader(imageBytes)
	contentType := http.DetectContentType(imageBytes[:512])
	if !strings.HasPrefix(contentType, "image") && !filetype.IsImage(imageBytes[:300]) {
		if err != nil {
			return err
		}
		fmt.Printf("Cannot proceed with processing due to internal error, %v", err)
		// If file is not an image, do not return an error to prevent lambda from being rerun.
		return nil
	}
	destinationRoot := getCatalogueFilePath(fileKey)

	// Copy root file to output bucket
	err = copyData(fileKey, destinationRoot+"/1080.jpg", contentType)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	// Save 720 version
	err = ResizeImage(bytesReader, &out, VideoSizes["720p"])
	if err != nil {
		return err
	}
	completeRequest(&out, contentType, destinationRoot+"/720.jpg")
	// Save 200 version
	out.Reset()
	bytesReader.Seek(0, 0)
	err = ResizeImage(bytesReader, &out, Dimension{200, 200})
	if err != nil {
		return err
	}
	completeRequest(&out, contentType, destinationRoot+"/200.jpg")

	return nil
}

// ResizeImage resizes the provided image to a destination dimension
func ResizeImage(input io.Reader, output io.Writer, d Dimension) error {
	width, height := strconv.Itoa(d.width), strconv.Itoa(d.height)
	cmd := exec.Command("ffmpeg",
		"-i", "pipe:0",
		"-f", "image2",
		"-vf", fmt.Sprintf("scale=%s:%s:force_original_aspect_ratio=decrease,pad=%s:%s:(ow-iw)/2:(oh-ih)/2", width, height, width, height),
		"pipe:1",
	)

	cmd.Stdin = input
	cmd.Stdout = output

	err := cmd.Run()
	if err != nil {
		return err
	}

	return err
}
