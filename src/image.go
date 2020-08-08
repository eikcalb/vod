package vod

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/h2non/filetype"
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

		completeRequest(&out, contentType, getCatalogueFilePath()+"/600")
		c.JSON(http.StatusOK, gin.H{"message": "Successfully processed data"})
	})
	log.Print(g)
	return g
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
