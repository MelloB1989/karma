package s3

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/MelloB1989/karma/utils"

	c "github.com/MelloB1989/karma/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

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
