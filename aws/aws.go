package aws

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var svc *s3.S3

func InitializeAWSClient() error {
	region := os.Getenv("S3_BUCKET_REGION")
	accessKey := os.Getenv("S3_ACCESS_KEY_ID")
	secretKey := os.Getenv("S3_SECRET_ACCESS_KEY")

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return err
	}

	svc = s3.New(sess)

	return nil
}

func GetChannelIDFromS3() (string, error) {
	bucket := os.Getenv("S3_BUCKET_NAME")
	file := os.Getenv("S3_FILE_NAME")

	// Downloading the file
	result, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
	})

	if err != nil {
		// Check if the error is due to the file not existing
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return "1", nil
			default:
				return "", fmt.Errorf("Error fetching file: %s", err)
			}
		} else {
			return "", fmt.Errorf("Error fetching file: %s", err)
		}
	}

	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		return "", fmt.Errorf("Error reading file: %s", err)
	}

	// Convert content to int
	id, err := strconv.Atoi(string(body))
	if err != nil {
		return "", fmt.Errorf("Error converting string to int: %s", err)
	}

	// Increment ID as it is the ID of the last used channel
	stringID := strconv.Itoa(id + 1)
	return stringID, nil
}

func IncrementChannelIDInS3() error {
	currentID, err := GetChannelIDFromS3()
	if err != nil {
		return fmt.Errorf("Failed to get channel ID from S3 with error: %s", err)
	}

	bucket := os.Getenv("S3_BUCKET_NAME")
	file := os.Getenv("S3_FILE_NAME")

	// Upload incremented ID back to S3
	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
		Body:   bytes.NewReader([]byte(currentID)),
	})
	if err != nil {
		return fmt.Errorf("Error uploading new channel ID to S3 with error: %s,", err)
	}

	return nil
}
