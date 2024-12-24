package files

import (
	"bytes"
	"fmt"
	"mime/multipart"

	"github.com/MelloB1989/karma/apis/aws/s3"
	"github.com/MelloB1989/karma/config"
	"github.com/gofiber/fiber/v2"
)

func HandleSingleFileUpload(file fiber.FormFile, prefix string) (string, error) {
	fileReader := bytes.NewReader(file.Content)
	file.Name = prefix + "/" + file.Name
	err := s3.UploadRawFile(file.Name, fileReader)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s/%s", config.DefaultConfig().AwsBucketName, prefix, file.Name), nil
}

func HandleMultipleFileUpload(files []*multipart.FileHeader, prefix string) ([]string, error) {
	urls := []string{}
	for _, file := range files {
		f, err := file.Open()
		if err != nil {
			return urls, err
		}
		defer f.Close()
		err = s3.UploadRawFile(file.Filename, f)
		if err != nil {
			return urls, err
		}
		urls = append(urls, fmt.Sprintf("https://%s.s3.amazonaws.com/%s/%s", config.DefaultConfig().AwsBucketName, prefix, file.Filename))
	}
	return urls, nil
}
