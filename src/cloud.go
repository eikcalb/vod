package vod

import (
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func downloadData(inputKey string, result io.WriterAt, bucket string) error {
	awsID := os.Getenv(Config.AWS.AccessKeyID)
	awsSecret := os.Getenv(Config.AWS.AccessKeySecret)
	creds := credentials.NewStaticCredentials(awsID, awsSecret, "")

	config := &aws.Config{
		Credentials: creds,
		Region:      aws.String(Config.AWS.Region),
	}
	sess := session.Must(session.NewSession(config))
	downloader := s3manager.NewDownloader(sess)
	_, err := downloader.Download(result, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(inputKey),
	})
	if err != nil {
		return err
	}
	return nil
}

func completeRequest(data io.Reader, contentType string, path string) error {
	awsID := os.Getenv(Config.AWS.AccessKeyID)
	awsSecret := os.Getenv(Config.AWS.AccessKeySecret)
	creds := credentials.NewStaticCredentials(awsID, awsSecret, "")
	log.Println("Start write to AWS")

	config := &aws.Config{
		Credentials: creds,
		Region:      aws.String(Config.AWS.Region),
	}
	sess := session.Must(session.NewSession(config))
	uploader := s3manager.NewUploader(sess)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(Config.AWS.OutputBucketName),
		Key:         aws.String(path),
		ACL:         aws.String("public-read"),
		Body:        data,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return err
	}
	return nil
}
