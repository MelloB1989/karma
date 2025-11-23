package openai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/MelloB1989/karma/models"
	"github.com/MelloB1989/karma/utils"
	"github.com/openai/openai-go/v2"
)

func GenImage(prompt, model, destination_dir string, com ...CompatibleOptions) (*models.AIImageResponse, error) {
	client := createClient(com...)
	ctx := context.Background()

	// Generate image URL
	image, err := client.Images.Generate(ctx, openai.ImageGenerateParams{
		Prompt:         prompt,
		Model:          openai.ImageModel(model),
		ResponseFormat: openai.ImageGenerateParamsResponseFormatURL,
		N:              openai.Int(1),
	})
	if err != nil {
		return nil, err
	}

	imageURL := image.Data[0].URL

	// Download the image
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destination_dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Generate random filename with appropriate extension
	randomName := utils.GenerateID(16)

	// Determine file extension from Content-Type header or URL
	var extension string
	contentType := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "png"):
		extension = ".png"
	case strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg"):
		extension = ".jpg"
	case strings.Contains(contentType, "webp"):
		extension = ".webp"
	default:
		// Fallback to PNG if content type is unclear
		extension = ".png"
	}

	filename := randomName + extension
	filepath := filepath.Join(destination_dir, filename)

	// Create the file
	file, err := os.Create(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy the image data to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}

	return &models.AIImageResponse{
		FilePath:       filepath,
		ImageHostedUrl: imageURL,
	}, nil
}
