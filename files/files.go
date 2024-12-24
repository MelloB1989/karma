package files

import (
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

func (kf *KarmaFiles) HandleSingleFileUpload(file *multipart.FileHeader) (string, error) {
	if kf.UploadMode == S3 {
		return f.HandleSingleFileUploadToS3(file, kf.PathPrefix)
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
