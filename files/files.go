package files

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/MelloB1989/karma/apis/aws/s3"
	c "github.com/MelloB1989/karma/config"
	f "github.com/MelloB1989/karma/internal/files"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
)

type UploadModes string

const (
	S3            UploadModes = "S3"
	LOCAL         UploadModes = "LOCAL"
	DIGITAL_OCEAN UploadModes = "DIGITAL_OCEAN"
)

// FileVisibility represents the access level of uploaded files
type FileVisibility string

const (
	// Public makes the file publicly accessible via URL
	Public FileVisibility = "public"
	// Private makes the file accessible only via signed URLs or with credentials
	Private FileVisibility = "private"
)

// UploadOptions contains options for file uploads
type UploadOptions struct {
	// Visibility determines if the file is public or private (default: Public)
	Visibility FileVisibility
	// NoFilePrefix if true, doesn't add a unique prefix to the filename
	NoFilePrefix bool
}

// DefaultUploadOptions returns the default upload options (public visibility)
func DefaultUploadOptions() UploadOptions {
	return UploadOptions{
		Visibility:   Public,
		NoFilePrefix: false,
	}
}

type KarmaFiles struct {
	PathPrefix     string
	UploadMode     UploadModes
	LocalUploadDir string
}

func NewKarmaFile(pathPrefix string, uploadMode UploadModes) *KarmaFiles {
	return &KarmaFiles{
		PathPrefix: pathPrefix,
		UploadMode: uploadMode,
	}
}

func BytesToMultipartFileHeader(imageBytes []byte, filename string) (*multipart.FileHeader, error) {
	// Create a buffer to hold the multipart form data
	var buffer bytes.Buffer

	// Create a multipart writer to write the form data to the buffer
	writer := multipart.NewWriter(&buffer)

	// Create a form file part
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}

	// Write the image bytes to the form file part
	_, err = part.Write(imageBytes)
	if err != nil {
		return nil, err
	}

	// Close the multipart writer to finalize the form data
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	// Create a new multipart reader to parse the form data
	reader := multipart.NewReader(&buffer, writer.Boundary())

	// Parse the multipart form data
	form, err := reader.ReadForm(int64(buffer.Len()))
	if err != nil {
		return nil, err
	}

	// Get the file header from the parsed form data
	fileHeaders := form.File["file"]
	if len(fileHeaders) == 0 {
		return nil, fmt.Errorf("no file found in the form data")
	}

	return fileHeaders[0], nil
}

func (kf *KarmaFiles) HandleSingleFileUpload(file *multipart.FileHeader, options ...map[string]any) (string, error) {
	opts := make(map[string]any)
	if len(options) > 0 {
		opts = options[0]
	}
	if kf.UploadMode == S3 {
		return f.HandleSingleFileUploadToS3(file, kf.PathPrefix, opts)
	} else {
		return f.HandleSingleFileUploadToLocal(file, kf.LocalUploadDir)
	}
}

// HandleSingleFileUploadWithOptions uploads a single file with the specified options
func (kf *KarmaFiles) HandleSingleFileUploadWithOptions(file *multipart.FileHeader, uploadOpts UploadOptions) (string, error) {
	opts := map[string]any{
		"isPublic":     uploadOpts.Visibility == Public,
		"noFilePrefix": uploadOpts.NoFilePrefix,
	}
	if kf.UploadMode == S3 {
		return f.HandleSingleFileUploadToS3(file, kf.PathPrefix, opts)
	} else {
		return f.HandleSingleFileUploadToLocal(file, kf.LocalUploadDir)
	}
}

func (kf *KarmaFiles) HandleMultipleFileUpload(files []*multipart.FileHeader) ([]string, error) {
	if kf.UploadMode == S3 {
		return f.HandleMultipleFileUploadToS3(files, kf.PathPrefix)
	} else {
		return f.HandleMultipleFileUploadToLocal(files, kf.LocalUploadDir)
	}
}

// HandleMultipleFileUploadWithOptions uploads multiple files with the specified options
func (kf *KarmaFiles) HandleMultipleFileUploadWithOptions(files []*multipart.FileHeader, uploadOpts UploadOptions) ([]string, error) {
	opts := map[string]any{
		"isPublic": uploadOpts.Visibility == Public,
	}
	if kf.UploadMode == S3 {
		return f.HandleMultipleFileUploadToS3(files, kf.PathPrefix, opts)
	} else {
		return f.HandleMultipleFileUploadToLocal(files, kf.LocalUploadDir)
	}
}

// GetFileSignedURL generates a signed URL for accessing a file based on the upload mode.
// For S3, it generates a presigned URL with the specified expiration time.
// For LOCAL, it returns a file:// URL or serves through a local HTTP endpoint.
// The objectKey is the path/key of the file (for S3) or the file path (for LOCAL).
// The expiration parameter specifies how long the signed URL should be valid.
func (kf *KarmaFiles) GetFileSignedURL(objectKey string, expiration time.Duration) (string, error) {
	switch kf.UploadMode {
	case S3:
		return kf.getS3SignedURL(objectKey, expiration)
	case LOCAL:
		return kf.getLocalFileURL(objectKey)
	default:
		return "", fmt.Errorf("unsupported upload mode: %s", kf.UploadMode)
	}
}

// getS3SignedURL generates a presigned URL for an S3 object
func (kf *KarmaFiles) getS3SignedURL(objectKey string, expiration time.Duration) (string, error) {
	// Create S3 client config
	clientConfig := s3.S3ClientConfig{
		Region: c.DefaultConfig().S3BucketRegion,
	}

	// Create S3 client
	s3Client, err := s3.CreateS3Client(clientConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Create presign client
	presignClient := s3sdk.NewPresignClient(s3Client)

	// Get bucket name from config
	bucketName := c.DefaultConfig().AwsBucketName

	// Generate presigned URL
	presignedReq, err := presignClient.PresignGetObject(context.TODO(), &s3sdk.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, s3sdk.WithPresignExpires(expiration))

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedReq.URL, nil
}

// getLocalFileURL returns a URL for accessing a local file
func (kf *KarmaFiles) getLocalFileURL(filePath string) (string, error) {
	// Construct the full path
	fullPath := filePath
	if kf.LocalUploadDir != "" && !filepath.IsAbs(filePath) {
		fullPath = filepath.Join(kf.LocalUploadDir, filePath)
	}

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", fullPath)
	}

	// Return file:// URL for local files
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return "file://" + absPath, nil
}

// GetFileSignedURLWithOptions generates a signed URL with additional options
type SignedURLOptions struct {
	Expiration         time.Duration
	ContentType        string
	ContentDisposition string
}

// GetFileSignedURLWithOptions generates a signed URL for accessing a file with additional options.
// For S3, it supports setting response content type and content disposition.
func (kf *KarmaFiles) GetFileSignedURLWithOptions(objectKey string, opts SignedURLOptions) (string, error) {
	if opts.Expiration == 0 {
		opts.Expiration = 15 * time.Minute // Default expiration
	}

	switch kf.UploadMode {
	case S3:
		return kf.getS3SignedURLWithOptions(objectKey, opts)
	case LOCAL:
		return kf.getLocalFileURL(objectKey)
	default:
		return "", fmt.Errorf("unsupported upload mode: %s", kf.UploadMode)
	}
}

// getS3SignedURLWithOptions generates a presigned URL for an S3 object with additional options
func (kf *KarmaFiles) getS3SignedURLWithOptions(objectKey string, opts SignedURLOptions) (string, error) {
	// Create S3 client config
	clientConfig := s3.S3ClientConfig{
		Region: c.DefaultConfig().S3BucketRegion,
	}

	// Create S3 client
	s3Client, err := s3.CreateS3Client(clientConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Create presign client
	presignClient := s3sdk.NewPresignClient(s3Client)

	// Get bucket name from config
	bucketName := c.DefaultConfig().AwsBucketName

	// Build GetObjectInput with optional parameters
	input := &s3sdk.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	if opts.ContentType != "" {
		input.ResponseContentType = aws.String(opts.ContentType)
	}

	if opts.ContentDisposition != "" {
		input.ResponseContentDisposition = aws.String(opts.ContentDisposition)
	}

	// Generate presigned URL
	presignedReq, err := presignClient.PresignGetObject(context.TODO(), input, s3sdk.WithPresignExpires(opts.Expiration))

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedReq.URL, nil
}

// ServeLocalFile serves a local file over HTTP (useful for local upload mode)
// This can be used to set up a simple file server endpoint
func (kf *KarmaFiles) ServeLocalFile(w http.ResponseWriter, r *http.Request, filePath string) error {
	if kf.UploadMode != LOCAL {
		return fmt.Errorf("ServeLocalFile is only supported for LOCAL upload mode")
	}

	// Construct the full path
	fullPath := filePath
	if kf.LocalUploadDir != "" && !filepath.IsAbs(filePath) {
		fullPath = filepath.Join(kf.LocalUploadDir, filePath)
	}

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return fmt.Errorf("file not found: %s", fullPath)
	}

	// Serve the file
	http.ServeFile(w, r, fullPath)
	return nil
}
