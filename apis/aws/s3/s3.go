package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"

	"github.com/MelloB1989/karma/utils"

	c "github.com/MelloB1989/karma/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type BucketBasics struct {
	S3Client *s3.Client
}

func UploadLargeFileToS3(accessKeyID, secretAccessKey, region, bucketName, objectKey, filePath string, debug bool) error {
	// Create a static credentials provider
	creds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""))

	// Load the AWS configuration with the provided credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		if debug {
			fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
			fmt.Println(err)
		}
		return err
	}

	// Create an S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Create a BucketBasics instance
	// basics := BucketBasics{S3Client: s3Client}

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

	// Set a reasonable part size (10MB as used in the example)
	var partMiBs int64 = 10

	// Create a custom reader that tracks progress
	reader := &ProgressReader{
		Reader:   file,
		Size:     fileSize,
		Progress: 0,
		Debug:    debug,
	}

	// Create an uploader with the progress reader
	uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
		u.PartSize = partMiBs * 1024 * 1024
		// Set a higher concurrency for faster uploads
		u.Concurrency = 5
	})

	// Upload the file to S3
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   reader,
		ACL:    types.ObjectCannedACLPublicRead,
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
		log.Printf("\nSuccessfully uploaded %s to %s/%s\n", filePath, bucketName, objectKey)
	}

	// Optionally, delete the local file after uploading
	// err = os.Remove(filePath)
	// if err != nil {
	// 	fmt.Println("Failed to delete file:", err)
	// 	return err
	// }

	return nil
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

func UploadFileToS3(accessKeyID, secretAccessKey, region, bucketName, objectKey, filePath string) error {
	// Create a static credentials provider
	creds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""))

	// Load the AWS configuration with the provided credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
		fmt.Println(err)
		return err
	}

	// Create an S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Open the file to upload
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Couldn't open file %v to upload. Here's why: %v\n", filePath, err)
		return err
	}
	defer file.Close()

	// Upload the file to S3
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   file,
		ACL:    "public-read",
	})
	if err != nil {
		log.Printf("Couldn't upload file %v to %v:%v. Here's why: %v\n", filePath, bucketName, objectKey, err)
		return err
	}

	// Optionally, delete the local file after uploading
	err = os.Remove(filePath)
	if err != nil {
		fmt.Println("Failed to delete file:", err)
		return err
	}

	return nil
}

func UploadFile(objectKey string, fileName string) error {
	bucketName := c.DefaultConfig().AwsBucketName
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	s3Config := aws.Config{
		Region:      *aws.String(c.DefaultConfig().S3BucketRegion),
		Credentials: sdkConfig.Credentials,
	}
	if err != nil {
		fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
		fmt.Println(err)
	}
	s3Client := s3.NewFromConfig(s3Config)
	file, err := os.Open(fileName)
	if err != nil {
		log.Printf("Couldn't open file %v to upload. Here's why: %v\n", fileName, err)
		return err
	}
	defer file.Close()

	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   file,
		ACL:    "public-read",
	})
	if err != nil {
		log.Printf("Couldn't upload file %v to %v:%v. Here's why: %v\n", fileName, bucketName, objectKey, err)
		return err
	}
	err = os.Remove(fileName)
	if err != nil {
		fmt.Println("Failed to delete file:", err)
		return err
	}
	return nil
}

func UploadRawFile(objectKey string, file multipart.File) (*string, error) {
	bucketName := c.DefaultConfig().AwsBucketName
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	s3Config := aws.Config{
		Region:      *aws.String(c.DefaultConfig().S3BucketRegion),
		Credentials: sdkConfig.Credentials,
	}
	if err != nil {
		fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
		fmt.Println(err)
	}

	s3Client := s3.NewFromConfig(s3Config)

	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   file,
		ACL:    "public-read",
	})
	if err != nil {
		log.Printf("Couldn't upload file %v to %v:%v. Here's why: %v\n", "0", bucketName, objectKey, err)
		return nil, err
	}
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, c.DefaultConfig().S3BucketRegion, objectKey)
	return aws.String(url), nil
}

func GetFileByPath(objectKey string) (*os.File, error) {
	bucketName := c.DefaultConfig().AwsBucketName
	destinationPath := "./tmp/" + utils.GenerateID()
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
		fmt.Println(err)
		return nil, err
	}

	s3Client := s3.NewFromConfig(sdkConfig)
	resp, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	outFile, err := os.Create(destinationPath)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		outFile.Close()
		return nil, err
	}

	_, err = outFile.Seek(0, io.SeekStart)
	if err != nil {
		outFile.Close()
		return nil, err
	}

	return outFile, nil
}
