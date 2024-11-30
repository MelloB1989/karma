package segmind

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"karmaclips/aws/s3"
	"karmaclips/config"
	"karmaclips/utils"
	"log"
	"net/http"
	"os"
)

func RequestCreateImage(prompt string, model string, batch_size int, width int, height int) (*string, error) {
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

	var api string
	switch model {
	case "sd":
		api = config.NewConfig().SegmindSDAPI
	case "protovis":
		api = config.NewConfig().SegmindProtovisAPI
	case "samaritan":
		api = config.NewConfig().SegmindSamaritanAPI
	case "dreamshaper":
		api = config.NewConfig().SegmindDreamshaperAPI
	default:
		api = config.NewConfig().SegmindSDAPI
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
	req.Header.Set("x-api-key", config.NewConfig().SegmindAPIKey)
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
	uri := fmt.Sprintf("https://%s.s3.ap-south-1.amazonaws.com/karmaclips/%s", config.NewConfig().AwsBucketName, fileId)
	return &uri, nil
}
