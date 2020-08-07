package vod

import (
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func completeRequest(data io.Reader, contentType string, path string) error {
	log.Printf("configuration is %v", Config)
	creds := credentials.NewStaticCredentials(os.Getenv(Config.AWS.AWS_ACCESS_KEY_ID), os.Getenv(Config.AWS.AWS_SECRET_ACCESS_KEY), "")
	log.Println("Start write to AWS")

	config := &aws.Config{
		Credentials: creds,
		Region:      aws.String("us-east-2"),
	}
	sess := session.Must(session.NewSession(config))
	uploader := s3manager.NewUploader(sess)
	resp, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String("vod-file-storage"),
		Key:         aws.String(path),
		ACL:         aws.String("public-read"),
		Body:        data,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return err
	}
	log.Printf("Upload successful to %s", resp.Location)
	return nil
}
