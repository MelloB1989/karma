package segmind

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/MelloB1989/karma/apis/aws/s3"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/utils"
)

type SegmindSecrets struct {
	SegmindAPIKey         string `env:"SEGMIND_API_KEY"`
	SegmindSDAPI          string `env:"SEGMIND_SD_API"`
	SegmindSamaritanAPI   string `env:"SEGMIND_SAMARITAN_API"`
	SegmindDreamshaperAPI string `env:"SEGMIND_DREAMSHAPER_API"`
	SegmindProtovisAPI    string `env:"SEGMIND_PROTOVIS_API"`
}

func RequestCreateImage(prompt string, model string, batch_size int, width int, height int, s3dir string) (*string, error) {
	data := map[string]interface{}{
		"prompt":          prompt,
		"negative_prompt": "low quality, blurry",
		"steps":           25,
		"guidance_scale":  5.5,
		"seed":            98552302,
		"sampler":         "euler",
		"scheduler":       "sgm_uniform",
		"width":           width,
		"height":          height,
		"aspect_ratio":    "custom",
		"batch_size":      batch_size,
		"image_format":    "jpeg",
		"image_quality":   95,
		"base64":          true,
	}

	segmindSecrets := &SegmindSecrets{}
	err := config.CustomConfig(segmindSecrets)
	if err != nil {
		log.Printf("Error loading Segmind secrets: %v", err)
		return nil, err
	}

	var api string
	switch model {
	case "sd":
		api = segmindSecrets.SegmindSDAPI
	case "protovis":
		api = segmindSecrets.SegmindProtovisAPI
	case "samaritan":
		api = segmindSecrets.SegmindSamaritanAPI
	case "dreamshaper":
		api = segmindSecrets.SegmindDreamshaperAPI
	default:
		api = segmindSecrets.SegmindSDAPI
	}

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
	req.Header.Set("x-api-key", segmindSecrets.SegmindAPIKey)
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
	fileId := utils.GenerateID() + ".jpeg"
	fileName := "./tmp/" + fileId
	err = os.WriteFile(fileName, imageBytes, 0644)
	if err != nil {
		fmt.Println("Error saving image file:", err)
		return nil, err
	}

	// Upload to S3
	err = s3.UploadFile("karmaclips/"+fileId, fileName)
	if err != nil {
		log.Printf("Error uploading image to S3: %v", err)
		return nil, err
	}

	// Clean up local file
	os.Remove(fileName)

	// Build the S3 URL
	uri := fmt.Sprintf("https://%s.s3.ap-south-1.amazonaws.com/%s/%s", config.DefaultConfig().AwsBucketName, s3dir, fileId)
	return &uri, nil
}
