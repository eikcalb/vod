package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	vod "eikcalb.dev/vod/src"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gin-gonic/gin"
)

func setupRouter(config *vod.Configuration) *gin.Engine {
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

func setupLambda(ctx context.Context, event events.S3Event) error {
	for _, record := range event.Records {
		log.Print(record.S3.Object.URLDecodedKey)
		switch record.S3.Bucket.Name {
		case vod.Config.AWS.CatalogueBucketName:
			err := vod.HandleAWSCatalogue(record.S3)
			if err != nil {
				log.Println(err)
				return err
			}
		case vod.Config.AWS.MediaBucketName:
			err := vod.HandleAWSMedia(record.S3)
			if err != nil {
				log.Println(err)
				return err
			}
		}
	}
	return nil
}

func serverMain() {
	r := setupRouter(vod.Config)

	// Register middleware routers
	vod.CreateVideoServer(r, vod.Config)
	vod.CreateImageServer(r)

	log.Printf("Starting %s server!\n========\tUsing address %s:%v\t=========", vod.Config.AppName, vod.Config.Listen.Host, vod.Config.Listen.Port)
	err := r.Run(fmt.Sprintf("%s:%s", vod.Config.Listen.Host, strconv.Itoa(vod.Config.Listen.Port)))
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	lambda.Start(setupLambda)
}
