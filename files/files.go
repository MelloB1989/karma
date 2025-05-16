package files

import (
	"bytes"
	"fmt"
	"mime/multipart"

	f "github.com/MelloB1989/karma/internal/files"
)

type UploadModes string

const (
	S3            UploadModes = "S3"
	LOCAL         UploadModes = "LOCAL"
	DIGITAL_OCEAN UploadModes = "DIGITAL_OCEAN"
)

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

func (kf *KarmaFiles) HandleMultipleFileUpload(files []*multipart.FileHeader) ([]string, error) {
	if kf.UploadMode == S3 {
		return f.HandleMultipleFileUploadToS3(files, kf.PathPrefix)
	} else {
		return f.HandleMultipleFileUploadToLocal(files, kf.LocalUploadDir)
	}
}
