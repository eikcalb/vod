package vod

import (
	"log"

	"github.com/gin-gonic/gin"
)

// CreateImageServer creates an image server
func CreateImageServer(r *gin.Engine) *gin.RouterGroup {
	g := r.Group("/image")

	log.Print(g)
	return g
}
