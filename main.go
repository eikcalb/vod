package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	vod "eikcalb.dev/vod/src"
	"github.com/gin-gonic/gin"
)

func SetupRouter(config *vod.Configuration) *gin.Engine {
	switch strings.ToLower(config.ServerMode) {
	case "release":
		gin.SetMode(gin.ReleaseMode)
	case "debug":
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()
	r.MaxMultipartMemory = config.MaxUploadSize

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	return r
}

func main() {
	r := SetupRouter(vod.Config)

	// Register middleware routers
	vod.CreateVideoServer(r, config)

	log.Printf("Starting %s server!", config.AppName)
	err = r.Run(fmt.Sprintf("%s:%s", config.Listen.Host, strconv.Itoa(config.Listen.Port)))
	if err != nil {
		log.Fatal(err)
	}
}
