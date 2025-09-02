package gemini

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/MelloB1989/karma/utils"
	"google.golang.org/genai"
)

func GenImage(prompt, model, destination_dir string) (*models.AIImageResponse, error) {

	ctx := context.Background()

	apiKey := config.GetEnvRaw("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set or empty")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	parts := []*genai.Part{
		genai.NewPartFromText(prompt),
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	result, err := client.Models.GenerateContent(
		ctx,
		model,
		contents,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("received nil result from GenerateContent")
	}

	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned from GenerateContent")
	}

	if result.Candidates[0].Content == nil {
		return nil, fmt.Errorf("candidate content is nil")
	}

	if result.Candidates[0].Content.Parts == nil {
		return nil, fmt.Errorf("candidate content parts is nil")
	}

	if err := os.MkdirAll(destination_dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var path string

	for _, part := range result.Candidates[0].Content.Parts {
		if part.Text != "" {
			fmt.Println(part.Text)
		} else if part.InlineData != nil {
			randomName := utils.GenerateID(16)
			imageBytes := part.InlineData.Data
			outputFilename := fmt.Sprintf("%s/%s.png", destination_dir, randomName)
			if err := os.WriteFile(outputFilename, imageBytes, 0644); err != nil {
				return nil, fmt.Errorf("failed to write image file: %w", err)
			}
			path = outputFilename
		}
	}

	return &models.AIImageResponse{
		FilePath: path,
	}, nil
}
