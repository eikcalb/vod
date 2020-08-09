package vod

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
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
		err = ResizeImage(buf, &out, NewDimension(600, 600))
		if err != nil {
			log.Printf(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot proceed with processing due to internal error"})
			return
		}

		completeRequest(&out, contentType, getCatalogueFilePath()+"/600.png")
		c.JSON(http.StatusOK, gin.H{"message": "Successfully processed data"})
	})

	return g
}

// HandleAWSCatalogue is called in lambda upon activity in a lambda
func HandleAWSCatalogue(s3 events.S3Entity) error {
	inputData := aws.NewWriteAtBuffer([]byte{})
	log.Printf("Data URL decoded key is %s\nNormal key is %s", s3.Object.URLDecodedKey, s3.Object.Key)
	err := downloadData(s3.Object.Key, inputData, Config.AWS.InputBucketName)
	if err != nil {
		return err
	}

	buf := bufio.NewReaderSize(bytes.NewReader(inputData.Bytes()), 600)
	head, err := buf.Peek(512)
	contentType := http.DetectContentType(head)
	if err != nil || (!strings.HasPrefix(contentType, "image") && !filetype.IsImage(head)) {
		if err != nil {
			return err
		}
		return errors.New("Cannot proceed with processing due to internal error")

	}
	var out bytes.Buffer
	err = ResizeImage(buf, &out, NewDimension(600, 600))
	if err != nil {
		return err
	}

	completeRequest(&out, contentType, getCatalogueFilePath()+"/600.png")
	return nil
}

// ResizeImage resizes the provided image to a destination dimension
func ResizeImage(input io.Reader, output io.Writer, d *Dimension) error {
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
