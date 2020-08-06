package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	path, err := filepath.Abs("config.json")
	if err != nil {
		log.Fatal("Cannot continue with application", err)
		os.Exit(1)
	}
	config := vod.LoadConfig(path)

	r := SetupRouter(config)

	// Register middleware routers
	vod.CreateVideoServer(r, config)

	log.Printf("Starting %s server!", config.AppName)
	err = r.Run(fmt.Sprintf("%s:%s", config.Listen.Host, strconv.Itoa(config.Listen.Port)))
	if err != nil {
		log.Fatal(err)
	}
}
