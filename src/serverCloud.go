package vod

import (
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	// AWSSession contains the AWS session variable
	AWSSession *session.Session
)

func downloadData(inputKey string, result io.WriterAt, bucket string) error {
	downloader := s3manager.NewDownloader(AWSSession)
	_, err := downloader.Download(result, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(inputKey),
	})
	if err != nil {
		return err
	}
	return nil
}

func copyData(inputKey, outputKey, contentType string) error {
	client := s3.New(AWSSession)
	_, err := client.CopyObject(&s3.CopyObjectInput{
		Bucket:      aws.String(Config.AWS.OutputBucketName),
		CopySource:  aws.String(Config.AWS.InputBucketName + "/" + inputKey),
		Key:         aws.String(outputKey),
		ACL:         aws.String("public-read"),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return err
	}
	// No need to wait for copy to complete
	//err = svc.WaitUntilObjectExists(&s3.HeadObjectInput{Bucket: aws.String(Config.AWS.OutputBucketName), Key: aws.String(outputKey)})
	return nil
}

func completeRequest(data io.Reader, contentType string, path string) error {
	uploader := s3manager.NewUploader(AWSSession)
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

func init() {
	awsID := os.Getenv(Config.AWS.AccessKeyID)
	awsSecret := os.Getenv(Config.AWS.AccessKeySecret)
	creds := credentials.NewStaticCredentials(awsID, awsSecret, "")

	config := &aws.Config{
		Credentials: creds,
		Region:      aws.String(Config.AWS.Region),
	}
	sess := session.Must(session.NewSession(config))
	AWSSession = sess
}
