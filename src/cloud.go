package vod

import (
	"io"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func completeRequest(data io.Writer) {
	creds := credentials.NewEnvCredentials()
	log.Println("Start write to AWS")

	config := &aws.Config{
		Credentials: creds,
		Region:      aws.String("us-east-2"),
	}
	sess := session.Must(session.NewSession(config))
	uploader := s3manager.NewUploader(sess)
	uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("vod-file-storage"),
	})
}
