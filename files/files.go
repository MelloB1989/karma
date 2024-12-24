package files

import (
	"mime/multipart"

	f "github.com/MelloB1989/karma/internal/files"
)

type KarmaFiles struct {
	PathPrefix     string
	BucketOverride string
}

func NewKarmaFile(pathPrefix, bucketOverride string) *KarmaFiles {
	return &KarmaFiles{
		PathPrefix:     pathPrefix,
		BucketOverride: bucketOverride,
	}
}

func (kf *KarmaFiles) HandleSingleFileUpload(file *multipart.FileHeader) (string, error) {
	return f.HandleSingleFileUpload(file, kf.PathPrefix)
}

func (kf *KarmaFiles) HandleMultipleFileUpload(files []*multipart.FileHeader) ([]string, error) {
	return f.HandleMultipleFileUpload(files, kf.PathPrefix)
}