package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"

	c "github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// S3ClientConfig holds configuration for creating an S3 client
type S3ClientConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	EnvPrefix       string
}

// getContentType determines the content type based on file extension
func getContentType(filename string) string {
	ext := filepath.Ext(filename)
	contentType := mime.TypeByExtension(ext)

	// If mime.TypeByExtension doesn't find it, use common mappings
	if contentType == "" {
		switch ext {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".gif":
			contentType = "image/gif"
		case ".webp":
			contentType = "image/webp"
		case ".svg":
			contentType = "image/svg+xml"
		case ".pdf":
			contentType = "application/pdf"
		case ".json":
			contentType = "application/json"
		case ".xml":
			contentType = "application/xml"
		case ".txt":
			contentType = "text/plain"
		case ".html", ".htm":
			contentType = "text/html"
		case ".css":
			contentType = "text/css"
		case ".js":
			contentType = "application/javascript"
		case ".mp4":
			contentType = "video/mp4"
		case ".mp3":
			contentType = "audio/mpeg"
		case ".zip":
			contentType = "application/zip"
		default:
			contentType = "application/octet-stream"
		}
	}

	return contentType
}

// CreateS3Client creates an S3 client with flexible configuration
func CreateS3Client(s3config S3ClientConfig) (*s3.Client, error) {
	// Determine region and credentials
	region := s3config.Region
	if region == "" {
		// Try to get region from environment variable with optional prefix
		envRegionKey := "AWS_REGION"
		if s3config.EnvPrefix != "" {
			envRegionKey = s3config.EnvPrefix + "_AWS_REGION"
		}
		region = os.Getenv(envRegionKey)
	}

	// Prepare credential options
	var credOptions []func(*config.LoadOptions) error

	// If region is specified, add it to config
	if region != "" {
		credOptions = append(credOptions, config.WithRegion(region))
	}

	// Check for credentials
	accessKeyID := s3config.AccessKeyID
	secretAccessKey := s3config.SecretAccessKey

	if accessKeyID == "" || secretAccessKey == "" {
		// Try to get from environment variables with optional prefix
		accessKeyEnvKey := "AWS_ACCESS_KEY_ID"
		secretKeyEnvKey := "AWS_SECRET_ACCESS_KEY"

		if s3config.EnvPrefix != "" {
			accessKeyEnvKey = s3config.EnvPrefix + "_AWS_ACCESS_KEY_ID"
			secretKeyEnvKey = s3config.EnvPrefix + "_AWS_SECRET_ACCESS_KEY"
		}

		accessKeyID = os.Getenv(accessKeyEnvKey)
		secretAccessKey = os.Getenv(secretKeyEnvKey)
	}

	// If specific credentials are provided, use them
	if accessKeyID != "" && secretAccessKey != "" {
		creds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""))
		credOptions = append(credOptions, config.WithCredentialsProvider(creds))
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(), credOptions...)
	if err != nil {
		return nil, fmt.Errorf("couldn't load AWS configuration: %v", err)
	}

	// Create and return S3 client
	return s3.NewFromConfig(cfg), nil
}

// ProgressReader is a custom io.Reader that tracks upload progress
type ProgressReader struct {
	Reader   io.Reader
	Size     int64
	Progress int64
	Debug    bool
}

// Read reads data from the underlying reader and updates progress
func (r *ProgressReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if n > 0 {
		r.Progress += int64(n)
		if r.Debug {
			percent := float64(r.Progress) / float64(r.Size) * 100

			// Display progress bar
			width := 50
			completed := int(float64(width) * float64(r.Progress) / float64(r.Size))

			fmt.Printf("\r[")
			for i := 0; i < width; i++ {
				if i < completed {
					fmt.Print("=")
				} else {
					fmt.Print(" ")
				}
			}

			fmt.Printf("] %.2f%% (%d/%d bytes)", percent, r.Progress, r.Size)
		}
	}
	return n, err
}

// UploadLargeFileToS3 uploads a large file to S3 with progress tracking
func UploadLargeFileToS3(opts S3ClientConfig, bucketName, objectKey, filePath string, debug bool) error {
	// Create S3 client
	s3Client, err := CreateS3Client(opts)
	if err != nil {
		if debug {
			fmt.Println("Couldn't create S3 client:", err)
		}
		return err
	}

	// Open the file to upload
	file, err := os.Open(filePath)
	if err != nil {
		if debug {
			log.Printf("Couldn't open file %v to upload. Here's why: %v\n", filePath, err)
		}
		return err
	}
	defer file.Close()

	// Get file size for progress reporting
	fileInfo, err := file.Stat()
	if err != nil {
		if debug {
			log.Printf("Couldn't get file info for %v. Here's why: %v\n", filePath, err)
		}
		return err
	}
	fileSize := fileInfo.Size()

	// Set a reasonable part size (10MB)
	var partMiBs int64 = 10

	// Create a custom reader that tracks progress
	reader := &ProgressReader{
		Reader:   file,
		Size:     fileSize,
		Progress: 0,
		Debug:    debug,
	}

	// Determine content type from file extension
	contentType := getContentType(filePath)

	// Create an uploader with the progress reader
	uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
		u.PartSize = partMiBs * 1024 * 1024
		// Set a higher concurrency for faster uploads
		u.Concurrency = 5
	})

	// Upload the file to S3
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(objectKey),
		Body:        reader,
		ACL:         types.ObjectCannedACLPublicRead,
		ContentType: aws.String(contentType),
	})

	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "EntityTooLarge" {
			if debug {
				log.Printf("Error while uploading object to %s. The object is too large.\n"+
					"The maximum size for a multipart upload is 5TB.", bucketName)
			}
		} else {
			if debug {
				log.Printf("Couldn't upload file %v to %v:%v. Here's why: %v\n",
					filePath, bucketName, objectKey, err)
			}
		}
		return err
	}

	if debug {
		log.Printf("\nSuccessfully uploaded %s to %s/%s with content type %s\n", filePath, bucketName, objectKey, contentType)
	}

	return nil
}

// UploadFile uploads a file to the default S3 bucket
func UploadFile(objectKey, fileName string, envPrefix ...string) error {
	prefix := ""
	if len(envPrefix) > 0 {
		prefix = envPrefix[0]
	}

	// Prepare S3 client config
	clientConfig := S3ClientConfig{
		Region:    c.DefaultConfig().S3BucketRegion,
		EnvPrefix: prefix,
	}

	// Create S3 client
	s3Client, err := CreateS3Client(clientConfig)
	if err != nil {
		fmt.Println("Couldn't create S3 client:", err)
		return err
	}

	// Open the file to upload
	file, err := os.Open(fileName)
	if err != nil {
		log.Printf("Couldn't open file %v to upload. Here's why: %v\n", fileName, err)
		return err
	}
	defer file.Close()

	// Determine content type from file extension
	contentType := getContentType(fileName)

	// Upload to default bucket
	bucketName := c.DefaultConfig().AwsBucketName
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(objectKey),
		Body:        file,
		ACL:         "public-read",
		ContentType: aws.String(contentType),
	})
	if err != nil {
		log.Printf("Couldn't upload file %v to %v:%v. Here's why: %v\n", fileName, bucketName, objectKey, err)
		return err
	}

	// Optionally delete the local file
	err = os.Remove(fileName)
	if err != nil {
		fmt.Println("Failed to delete file:", err)
		return err
	}

	return nil
}

// UploadRawFile uploads a multipart file to S3
func UploadRawFile(objectKey string, file multipart.File, envPrefix ...string) (*string, error) {
	prefix := ""
	if len(envPrefix) > 0 {
		prefix = envPrefix[0]
	}

	// Prepare S3 client config
	clientConfig := S3ClientConfig{
		Region:    c.DefaultConfig().S3BucketRegion,
		EnvPrefix: prefix,
	}

	// Create S3 client
	s3Client, err := CreateS3Client(clientConfig)
	if err != nil {
		fmt.Println("Couldn't create S3 client:", err)
		return nil, err
	}

	// Determine content type from object key extension
	contentType := getContentType(objectKey)

	// Upload to default bucket
	bucketName := c.DefaultConfig().AwsBucketName
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(objectKey),
		Body:        file,
		ACL:         "public-read",
		ContentType: aws.String(contentType),
	})
	if err != nil {
		log.Printf("Couldn't upload file %v to %v:%v. Here's why: %v\n", "0", bucketName, objectKey, err)
		return nil, err
	}

	// Generate and return S3 URL
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, c.DefaultConfig().S3BucketRegion, objectKey)
	return aws.String(url), nil
}

// GetFileByPath downloads a file from S3 to a local temporary path
func GetFileByPath(objectKey string, envPrefix ...string) (*os.File, error) {
	prefix := ""
	if len(envPrefix) > 0 {
		prefix = envPrefix[0]
	}

	// Get bucket name and region from default config
	bucketName := c.DefaultConfig().AwsBucketName
	bucketRegion := c.DefaultConfig().S3BucketRegion

	// Prepare S3 client config with explicit region
	clientConfig := S3ClientConfig{
		Region:    bucketRegion,
		EnvPrefix: prefix,
	}

	// Create S3 client with the specific bucket region
	s3Client, err := CreateS3Client(clientConfig)
	if err != nil {
		fmt.Println("Couldn't create S3 client:", err)
		return nil, err
	}

	// Prepare download
	destinationPath := "/tmp/" + utils.GenerateID()

	// Download object
	resp, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		// Log the full error for debugging
		log.Printf("S3 GetObject error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Create local file
	outFile, err := os.Create(destinationPath)
	if err != nil {
		return nil, err
	}

	// Copy file contents
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		outFile.Close()
		return nil, err
	}

	// Reset file pointer
	_, err = outFile.Seek(0, io.SeekStart)
	if err != nil {
		outFile.Close()
		return nil, err
	}

	return outFile, nil
}
