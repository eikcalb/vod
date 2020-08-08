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
	vod.CreateVideoServer(r, vod.Config)
	vod.CreateImageServer(r)

	log.Printf("Starting %s server!\n========\tUsing address %s:%v\t=========", vod.Config.AppName, vod.Config.Listen.Host, vod.Config.Listen.Port)
	err := r.Run(fmt.Sprintf("%s:%s", vod.Config.Listen.Host, strconv.Itoa(vod.Config.Listen.Port)))
	if err != nil {
		log.Fatal(err)
	}
}
