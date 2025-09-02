package segmind

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/files"
	"github.com/MelloB1989/karma/utils"
)

type SegmindModels string

type Resolutions struct {
	Height int `json:"height"`
	Width  int `json:"width"`
}

const (
	SegmindSDAPI          SegmindModels = "https://api.segmind.com/v1/stable-diffusion-3.5-large-txt2img"
	SegmindProtovisAPI    SegmindModels = "https://api.segmind.com/v1/sdxl1.0-protovis-lightning"
	SegmindSamaritanAPI   SegmindModels = "https://api.segmind.com/v1/sdxl1.0-samaritan-3d"
	SegmindDreamshaperAPI SegmindModels = "https://api.segmind.com/v1/sdxl1.0-dreamshaper-lightning"
	SegmindNanoBananaAPI  SegmindModels = "https://api.segmind.com/v1/nano-banana"
	SegmindFluxAPI        SegmindModels = "https://api.segmind.com/v1/flux-schnell"
	SegmindMidjourneyAPI  SegmindModels = "https://api.segmind.com/v1/midjourney"
	SegmindSDXLAPI        SegmindModels = "https://api.segmind.com/v1/sdxl1.0-txt2img"
	SegmindSD15API        SegmindModels = "https://api.segmind.com/v1/sd1.5-txt2img"
)

var (
	R1024x1024 = Resolutions{Height: 1024, Width: 1024}
	R896x1152  = Resolutions{Height: 1152, Width: 896}
	R832x1216  = Resolutions{Height: 1216, Width: 832}
	R768x1344  = Resolutions{Height: 1344, Width: 768}
	R640x1536  = Resolutions{Height: 1536, Width: 640}
	R1216x832  = Resolutions{Height: 832, Width: 1216}
	R1344x768  = Resolutions{Height: 768, Width: 1344}
	R1536x640  = Resolutions{Height: 640, Width: 1536}
)

type Segmind struct {
	Model      SegmindModels
	ApiKey     string
	BatchSize  int
	Width      int
	Height     int
	OutputDir  string
	UploadToS3 bool
	S3Bucket   string
}

type Options func(*Segmind)

func NewSegmind(model SegmindModels, opts ...Options) *Segmind {
	r := R1024x1024
	k, _ := config.GetEnv("SEGMIND_API_KEY")
	return &Segmind{
		Model:      model,
		BatchSize:  1,
		Width:      r.Width,
		Height:     r.Height,
		OutputDir:  "./images",
		UploadToS3: false,
		S3Bucket:   "",
		ApiKey:     k,
	}
}

func WithBatchSize(batchSize int) Options {
	return func(s *Segmind) {
		s.BatchSize = batchSize
	}
}

func WithResolution(res Resolutions) Options {
	return func(s *Segmind) {
		s.Width = res.Width
		s.Height = res.Height
	}
}

func WithOutputDir(outputDir string) Options {
	return func(s *Segmind) {
		s.OutputDir = outputDir
	}
}

func WithS3Upload(bucket string) Options {
	return func(s *Segmind) {
		s.UploadToS3 = true
		s.S3Bucket = bucket
	}
}

func WithApiKey(apiKey string) Options {
	return func(s *Segmind) {
		s.ApiKey = apiKey
	}
}

func (s *Segmind) RequestCreateImage(prompt string) (*string, error) {
	data := map[string]any{
		"prompt":          prompt,
		"negative_prompt": "low quality, blurry",
		"steps":           25,
		"guidance_scale":  5.5,
		"seed":            98552302,
		"sampler":         "euler",
		"scheduler":       "sgm_uniform",
		"width":           s.Width,
		"height":          s.Height,
		"aspect_ratio":    "custom",
		"batch_size":      s.BatchSize,
		"image_format":    "jpeg",
		"image_quality":   95,
		"base64":          true,
	}

	api := string(s.Model)

	jsonPayload, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error converting struct to json:", err)
		return nil, err
	}
	req, err := http.NewRequest("POST", api, bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.ApiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	// Check if response is binary data (image) or JSON
	var imageBytes []byte

	// Try to parse as JSON first
	var responseData map[string]interface{}
	if json.Unmarshal(body, &responseData) == nil {
		// JSON response - extract base64 image
		if imageData, ok := responseData["image"].(string); ok {
			// Decode the base64 image data
			imageBytes, err = base64.StdEncoding.DecodeString(imageData)
			if err != nil {
				fmt.Println("Error decoding base64 image data:", err)
				return nil, err
			}
		} else {
			fmt.Println("No image data found in JSON response")
			return nil, fmt.Errorf("no image data found in response")
		}
	} else {
		// Binary response - use raw body as image data
		imageBytes = body
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(s.OutputDir, 0755); err != nil {
		fmt.Printf("Warning: Could not create output directory: %v\n", err)
	}

	// Generate filename and save locally
	fileId := utils.GenerateID() + ".jpeg"
	localFilePath := filepath.Join(s.OutputDir, fileId)

	// Save image locally
	err = os.WriteFile(localFilePath, imageBytes, 0644)
	if err != nil {
		fmt.Printf("Error saving image locally: %v\n", err)
		return nil, err
	}

	fmt.Printf("Image saved locally: %s\n", localFilePath)

	// Try to upload to S3 if enabled
	if s.UploadToS3 && s.S3Bucket != "" {
		kf := files.NewKarmaFile(s.S3Bucket, files.S3)
		image, err := files.BytesToMultipartFileHeader(imageBytes, "Karma Imager")
		if err != nil {
			fmt.Printf("Warning: Error converting image bytes to file for S3 upload: %v\n", err)
			fmt.Printf("Falling back to local file: %s\n", localFilePath)
			return &localFilePath, nil
		}

		url, err := kf.HandleSingleFileUpload(image)
		if err != nil {
			fmt.Printf("Warning: Error uploading image to S3: %v\n", err)
			fmt.Printf("Falling back to local file: %s\n", localFilePath)
			return &localFilePath, nil
		}

		fmt.Printf("Image uploaded to S3: %s\n", url)
		return &url, nil
	}

	// Return local file path
	return &localFilePath, nil
}

func (s *Segmind) RequestCreateImageWithInputImage(prompt string, imageUrls []string) (*string, error) {
	// Nano Banana requires at least one input image
	if s.Model == SegmindNanoBananaAPI && len(imageUrls) == 0 {
		return nil, fmt.Errorf("nano-banana model requires at least one input image URL")
	}

	data := map[string]interface{}{
		"prompt":     prompt,
		"image_urls": imageUrls,
	}

	api := string(s.Model)

	jsonPayload, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error converting struct to json:", err)
		return nil, err
	}
	req, err := http.NewRequest("POST", api, bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.ApiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	// Check if response is binary data (image) or JSON
	var imageBytes []byte

	// Try to parse as JSON first
	var responseData map[string]interface{}
	if json.Unmarshal(body, &responseData) == nil {
		// JSON response - extract base64 image
		if imageData, ok := responseData["image"].(string); ok {
			// Decode the base64 image data
			imageBytes, err = base64.StdEncoding.DecodeString(imageData)
			if err != nil {
				fmt.Println("Error decoding base64 image data:", err)
				return nil, err
			}
		} else {
			fmt.Println("No image data found in JSON response")
			return nil, fmt.Errorf("no image data found in response")
		}
	} else {
		// Binary response - use raw body as image data
		imageBytes = body
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(s.OutputDir, 0755); err != nil {
		fmt.Printf("Warning: Could not create output directory: %v\n", err)
	}

	// Generate filename and save locally
	fileId := utils.GenerateID() + ".jpeg"
	localFilePath := filepath.Join(s.OutputDir, fileId)

	// Save image locally
	err = os.WriteFile(localFilePath, imageBytes, 0644)
	if err != nil {
		fmt.Printf("Error saving image locally: %v\n", err)
		return nil, err
	}

	fmt.Printf("Image saved locally: %s\n", localFilePath)

	// Try to upload to S3 if enabled
	if s.UploadToS3 && s.S3Bucket != "" {
		kf := files.NewKarmaFile(s.S3Bucket, files.S3)
		image, err := files.BytesToMultipartFileHeader(imageBytes, "Karma Imager")
		if err != nil {
			fmt.Printf("Warning: Error converting image bytes to file for S3 upload: %v\n", err)
			fmt.Printf("Falling back to local file: %s\n", localFilePath)
			return &localFilePath, nil
		}

		url, err := kf.HandleSingleFileUpload(image)
		if err != nil {
			fmt.Printf("Warning: Error uploading image to S3: %v\n", err)
			fmt.Printf("Falling back to local file: %s\n", localFilePath)
			return &localFilePath, nil
		}

		fmt.Printf("Image uploaded to S3: %s\n", url)
		return &url, nil
	}

	// Return local file path
	return &localFilePath, nil
}
