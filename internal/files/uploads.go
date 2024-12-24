package files

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/MelloB1989/karma/apis/aws/s3"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/utils"
)

func HandleSingleFileUploadToS3(file *multipart.FileHeader, prefix string) (string, error) {
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

func HandleMultipleFileUploadToS3(files []*multipart.FileHeader, prefix string) ([]string, error) {
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
