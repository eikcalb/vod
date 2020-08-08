package vod

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreateImageServer creates an image server
func CreateImageServer(r *gin.Engine) *gin.RouterGroup {
	g := r.Group("/image")

	g.POST("/", func(c *gin.Context) {

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
