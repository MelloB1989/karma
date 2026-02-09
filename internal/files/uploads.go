package files

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/MelloB1989/karma/apis/aws/s3"
	c "github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

// getACLFromOptions extracts the ACL setting from options map
// Returns ACLPublicRead by default, ACLPrivate if isPublic is explicitly set to false
func getACLFromOptions(opts map[string]any) s3.FileACL {
	if isPublic, ok := opts["isPublic"]; ok {
		if public, isBool := isPublic.(bool); isBool && !public {
			return s3.ACLPrivate
		}
	}
	return s3.ACLPublicRead
}

func HandleSingleFileUploadToS3(file *multipart.FileHeader, prefix string, options ...map[string]any) (string, error) {
	opts := map[string]any{}
	if len(options) > 0 {
		opts = options[0]
	}
	if opts["noFilePrefix"] == true {
		file.Filename = prefix + "/" + file.Filename
	} else {
		file.Filename = prefix + "/" + utils.GenerateID(25) + "_" + file.Filename
	}
	f, err := file.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Determine ACL from options
	acl := getACLFromOptions(opts)

	u, err := s3.UploadRawFileWithACL(file.Filename, f, acl)
	if err != nil {
		return "", err
	}
	return *u, nil
}

func HandleMultipleFileUploadToS3(files []*multipart.FileHeader, prefix string, options ...map[string]any) ([]string, error) {
	opts := map[string]any{}
	if len(options) > 0 {
		opts = options[0]
	}

	// Determine ACL from options
	acl := getACLFromOptions(opts)

	urls := []string{}
	for _, file := range files {
		f, err := file.Open()
		if err != nil {
			return urls, err
		}
		defer f.Close()
		file.Filename = prefix + "/" + utils.GenerateID(25) + "_" + file.Filename
		u, err := s3.UploadRawFileWithACL(file.Filename, f, acl)
		if err != nil {
			return urls, err
		}
		urls = append(urls, *u)
	}
	return urls, nil
}

func HandleSingleFileUploadToLocal(fileHeader *multipart.FileHeader, uploadDir string) (string, error) {
	srcFile, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("Failed to open file! %w", err)
	}
	defer srcFile.Close()

	err = os.Mkdir(uploadDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("Error creating directory, %w", err)
	}

	destFilePath := filepath.Join(uploadDir, fileHeader.Filename)

	destFile, err := os.Create(destFilePath)
	if err != nil {
		return "", fmt.Errorf("Error creating file, %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return "", fmt.Errorf("Error writing to file, %w", err)
	}
	defer destFile.Close()

	return destFilePath, nil
}

func HandleMultipleFileUploadToLocal(fileHeaders []*multipart.FileHeader, uploadDir string) ([]string, error) {
	s := []string{}
	for _, f := range fileHeaders {
		p, err := HandleSingleFileUploadToLocal(f, uploadDir)
		if err != nil {
			return s, err
		}
		s = append(s, p)
	}
	return s, nil
}

// GetSignedURLFromS3 generates a presigned URL for accessing an S3 object
// objectKey is the key/path of the object in S3
// expiration is the duration for which the URL will be valid
func GetSignedURLFromS3(objectKey string, expiration time.Duration, envPrefix ...string) (string, error) {
	prefix := ""
	if len(envPrefix) > 0 {
		prefix = envPrefix[0]
	}

	// Get bucket name and region from default config
	bucketName := c.DefaultConfig().AwsBucketName
	bucketRegion := c.DefaultConfig().S3BucketRegion

	// Prepare S3 client config
	clientConfig := s3.S3ClientConfig{
		Region:    bucketRegion,
		EnvPrefix: prefix,
	}

	// Create S3 client
	s3Client, err := s3.CreateS3Client(clientConfig)
	if err != nil {
		return "", fmt.Errorf("couldn't create S3 client: %w", err)
	}

	// Create presign client
	presignClient := awss3.NewPresignClient(s3Client)

	// Generate presigned URL
	presignedReq, err := presignClient.PresignGetObject(context.TODO(), &awss3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, awss3.WithPresignExpires(expiration))
	if err != nil {
		return "", fmt.Errorf("couldn't generate presigned URL: %w", err)
	}

	return presignedReq.URL, nil
}

// GetSignedURLFromLocal returns the local file path or a URL for accessing local files
// For local files, this returns the file path that can be served by a local file server
// basePath is the base URL path where files are served (e.g., "/files" or "http://localhost:8080/files")
// filePath is the path to the file in the local upload directory
func GetSignedURLFromLocal(filePath string, basePath string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}

	// Get the filename from the path
	filename := filepath.Base(filePath)

	// Construct the URL
	if basePath == "" {
		// Return the absolute file path if no base path is provided
		return filePath, nil
	}

	// Ensure basePath doesn't end with a slash
	if basePath[len(basePath)-1] == '/' {
		basePath = basePath[:len(basePath)-1]
	}

	return fmt.Sprintf("%s/%s", basePath, filename), nil
}
