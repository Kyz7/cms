package utils

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

var (
	S3Session       *session.Session
	S3Bucket        string
	S3Region        string
	CloudFrontURL   string
	UseLocalStorage bool = true // Toggle: true = local, false = S3
)

// InitS3 initializes S3 session (kept for future use)
func InitS3(bucket, region, cloudfrontURL string) error {
	S3Bucket = bucket
	S3Region = region
	CloudFrontURL = cloudfrontURL

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return err
	}

	S3Session = sess
	UseLocalStorage = false // Switch to S3 mode if initialized
	return nil
}

func UploadFile(file *multipart.FileHeader) (string, error) {
	if UseLocalStorage {
		return UploadToLocal(file)
	}
	return UploadToS3(file)
}

func UploadToS3(file *multipart.FileHeader) (string, error) {
	if S3Session == nil {
		return "", fmt.Errorf("S3 not initialized, using local storage instead")
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s/%s%s",
		time.Now().Format("2006/01"),
		uuid.New().String(),
		ext,
	)

	svc := s3.New(S3Session)

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(S3Bucket),
		Key:         aws.String(filename),
		Body:        src,
		ContentType: aws.String(file.Header.Get("Content-Type")),
		ACL:         aws.String("public-read"),
	})

	if err != nil {
		return "", err
	}

	if CloudFrontURL != "" {
		return CloudFrontURL + "/" + filename, nil
	}

	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
		S3Bucket, S3Region, filename), nil
}

func DeleteFile(url string) error {
	if UseLocalStorage {
		return DeleteFromLocal(url)
	}
	return DeleteFromS3(url)
}

func DeleteFromS3(url string) error {
	if S3Session == nil {
		return fmt.Errorf("S3 not initialized")
	}

	svc := s3.New(S3Session)
	key := extractKeyFromURL(url)

	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(S3Bucket),
		Key:    aws.String(key),
	})

	return err
}

func extractKeyFromURL(url string) string {
	// TODO: Implement based on your URL structure
	// Example: https://bucket.s3.region.amazonaws.com/2024/01/uuid.jpg
	// Extract: 2024/01/uuid.jpg
	return ""
}

func GetStorageMode() string {
	if UseLocalStorage {
		return "local"
	}
	return "s3"
}

func SetStorageMode(useLocal bool) {
	UseLocalStorage = useLocal
}
