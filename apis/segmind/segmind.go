package segmind

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/files"
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
	Model     SegmindModels
	ApiKey    string
	BatchSize int
	Width     int
	Height    int
	S3Dir     string
}

type Options func(*Segmind)

func NewSegmind(model SegmindModels, opts ...Options) *Segmind {
	r := R1024x1024
	k, _ := config.GetEnv("SEGMIND_API-KEY")
	return &Segmind{
		Model:     model,
		BatchSize: 1,
		Width:     r.Width,
		Height:    r.Height,
		S3Dir:     "",
		ApiKey:    k,
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

func WithS3Dir(s3dir string) Options {
	return func(s *Segmind) {
		s.S3Dir = s3dir
	}
}

func WithApiKey(apiKey string) Options {
	return func(s *Segmind) {
		s.ApiKey = apiKey
	}
}

func (s *Segmind) RequestCreateImage(prompt string) (*string, error) {
	data := map[string]interface{}{
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

	// Parse JSON response to get the base64 image data
	var responseData map[string]interface{}
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		fmt.Println("Error parsing JSON response:", err)
		return nil, err
	}

	// Get the base64 encoded image string
	imageData, ok := responseData["image"].(string)
	if !ok {
		fmt.Println("No image data found in response")
		return nil, err
	}

	// Decode the base64 image data
	imageBytes, err := base64.StdEncoding.DecodeString(imageData)
	if err != nil {
		fmt.Println("Error decoding base64 image data:", err)
		return nil, err
	}

	// Save the image to a file
	kf := files.NewKarmaFile(s.S3Dir, files.S3)
	image, err := files.BytesToMultipartFileHeader(imageBytes, "Karma Imager")
	if err != nil {
		fmt.Println("Error converting image bytes to file:", err)
		return nil, err
	}
	url, err := kf.HandleSingleFileUpload(image)
	if err != nil {
		fmt.Println("Error uploading image to S3:", err)
		return nil, err
	}
	// fileId := utils.GenerateID() + ".jpeg"
	// fileName := "./tmp/" + fileId
	// err = os.WriteFile(fileName, imageBytes, 0644)
	// if err != nil {
	// 	fmt.Println("Error saving image file:", err)
	// 	return nil, err
	// }

	// // Upload to S3
	// err = s3.UploadFile("karmaclips/"+fileId, fileName)
	// if err != nil {
	// 	log.Printf("Error uploading image to S3: %v", err)
	// 	return nil, err
	// }

	// // Clean up local file
	// os.Remove(fileName)

	return &url, nil
}
