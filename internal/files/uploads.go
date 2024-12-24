package files

import (
	"fmt"
	"mime/multipart"

	"github.com/MelloB1989/karma/apis/aws/s3"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/utils"
)

func HandleSingleFileUpload(file *multipart.FileHeader, prefix string) (string, error) {
	file.Filename = prefix + "/" + utils.GenerateID(25) + "_" + file.Filename
	f, err := file.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()
	err = s3.UploadRawFile(file.Filename, f)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", config.DefaultConfig().AwsBucketName, file.Filename), nil
}

func HandleMultipleFileUpload(files []*multipart.FileHeader, prefix string) ([]string, error) {
	urls := []string{}
	for _, file := range files {
		f, err := file.Open()
		if err != nil {
			return urls, err
		}
		defer f.Close()
		file.Filename = prefix + "/" + utils.GenerateID(25) + "_" + file.Filename
		err = s3.UploadRawFile(file.Filename, f)
		if err != nil {
			return urls, err
		}
		urls = append(urls, fmt.Sprintf("https://%s.s3.amazonaws.com/%s", config.DefaultConfig().AwsBucketName, file.Filename))
	}
	return urls, nil
}
