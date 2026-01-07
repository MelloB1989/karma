package gemini

import (
	"context"
	"fmt"
	"os"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/MelloB1989/karma/utils"
	"google.golang.org/genai"
)

// ImageGenConfig holds configuration for image generation
type ImageGenConfig struct {
	// API Key authentication (uses Gemini API backend)
	APIKey string
	// Vertex AI authentication
	ProjectID string
	Location  string
	// Image generation settings
	AspectRatio      string // e.g., "1:1", "16:9", "9:16", "4:3", "3:4"
	ImageSize        string // e.g., "1K", "2K"
	MimeType         string // e.g., "image/png", "image/jpeg"
	PersonGeneration string // e.g., "ALLOW_ALL", "BLOCK_ALL", "BLOCK_ONLY_ADULTS"
	Temperature      float32
	TopP             float32
	MaxOutputTokens  int32
	// Safety settings - set to true to disable safety filters
	DisableSafetyFilters bool
}

// ImageGenOption is a functional option for configuring image generation
type ImageGenOption func(*ImageGenConfig)

// WithAPIKey sets the API key for Gemini API backend
func WithAPIKey(apiKey string) ImageGenOption {
	return func(c *ImageGenConfig) {
		c.APIKey = apiKey
	}
}

// WithVertexAI sets the project ID and location for Vertex AI backend
func WithVertexAI(projectID, location string) ImageGenOption {
	return func(c *ImageGenConfig) {
		c.ProjectID = projectID
		c.Location = location
	}
}

// WithAspectRatio sets the aspect ratio for generated images
func WithAspectRatio(aspectRatio string) ImageGenOption {
	return func(c *ImageGenConfig) {
		c.AspectRatio = aspectRatio
	}
}

// WithImageSize sets the image size (e.g., "1K", "2K")
func WithImageSize(imageSize string) ImageGenOption {
	return func(c *ImageGenConfig) {
		c.ImageSize = imageSize
	}
}

// WithMimeType sets the output MIME type (e.g., "image/png", "image/jpeg")
func WithMimeType(mimeType string) ImageGenOption {
	return func(c *ImageGenConfig) {
		c.MimeType = mimeType
	}
}

// WithPersonGeneration sets the person generation policy
func WithPersonGeneration(policy string) ImageGenOption {
	return func(c *ImageGenConfig) {
		c.PersonGeneration = policy
	}
}

// WithTemperatureImg sets the temperature for image generation
func WithTemperatureImg(temperature float32) ImageGenOption {
	return func(c *ImageGenConfig) {
		c.Temperature = temperature
	}
}

// WithTopPImg sets the top-p for image generation
func WithTopPImg(topP float32) ImageGenOption {
	return func(c *ImageGenConfig) {
		c.TopP = topP
	}
}

// WithMaxOutputTokens sets the maximum output tokens
func WithMaxOutputTokens(maxTokens int32) ImageGenOption {
	return func(c *ImageGenConfig) {
		c.MaxOutputTokens = maxTokens
	}
}

// WithDisabledSafetyFilters disables all safety filters
func WithDisabledSafetyFilters() ImageGenOption {
	return func(c *ImageGenConfig) {
		c.DisableSafetyFilters = true
	}
}

// GenImage generates an image using the Gemini API with API key authentication
// This is the legacy function for backward compatibility
func GenImage(prompt, model, destination_dir string) (*models.AIImageResponse, error) {
	apiKey := config.GetEnvRaw("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set or empty")
	}

	return GenImageWithConfig(prompt, model, destination_dir, WithAPIKey(apiKey))
}

// GenImageWithConfig generates an image with custom configuration options
// Supports both Gemini API (with API key) and Vertex AI (with project/location) backends
func GenImageWithConfig(prompt, model, destination_dir string, opts ...ImageGenOption) (*models.AIImageResponse, error) {
	// Apply default configuration
	cfg := &ImageGenConfig{
		AspectRatio:      "1:1",
		ImageSize:        "1K",
		MimeType:         "image/png",
		PersonGeneration: "ALLOW_ALL",
		Temperature:      1.0,
		TopP:             0.95,
		MaxOutputTokens:  32768,
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	ctx := context.Background()

	// Create client based on configuration
	var client *genai.Client
	var err error

	if cfg.APIKey != "" {
		// Use Gemini API backend with API key
		client, err = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  cfg.APIKey,
			Backend: genai.BackendGeminiAPI,
		})
	} else if cfg.ProjectID != "" && cfg.Location != "" {
		// Use Vertex AI backend
		client, err = genai.NewClient(ctx, &genai.ClientConfig{
			Backend:  genai.BackendVertexAI,
			Project:  cfg.ProjectID,
			Location: cfg.Location,
		})
	} else {
		// Try environment variables for Vertex AI
		projectID := config.GetEnvRaw("GOOGLE_PROJECT_ID")
		location := config.GetEnvRaw("GOOGLE_LOCATION")
		if projectID != "" && location != "" {
			client, err = genai.NewClient(ctx, &genai.ClientConfig{
				Backend:  genai.BackendVertexAI,
				Project:  projectID,
				Location: location,
			})
		} else {
			// Fallback to API key from environment
			apiKey := config.GetEnvRaw("GEMINI_API_KEY")
			if apiKey == "" {
				return nil, fmt.Errorf("no authentication configured: set GEMINI_API_KEY or GOOGLE_PROJECT_ID/GOOGLE_LOCATION environment variables, or use WithAPIKey/WithVertexAI options")
			}
			client, err = genai.NewClient(ctx, &genai.ClientConfig{
				APIKey:  apiKey,
				Backend: genai.BackendGeminiAPI,
			})
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Build generation config with image settings
	genConfig := &genai.GenerateContentConfig{
		Temperature:        genai.Ptr(cfg.Temperature),
		TopP:               genai.Ptr(cfg.TopP),
		MaxOutputTokens:    cfg.MaxOutputTokens,
		ResponseModalities: []string{"TEXT", "IMAGE"},
	}

	// Add safety settings if filters should be disabled
	if cfg.DisableSafetyFilters {
		genConfig.SafetySettings = []*genai.SafetySetting{
			{
				Category:  genai.HarmCategoryHateSpeech,
				Threshold: genai.HarmBlockThresholdOff,
			},
			{
				Category:  genai.HarmCategoryDangerousContent,
				Threshold: genai.HarmBlockThresholdOff,
			},
			{
				Category:  genai.HarmCategorySexuallyExplicit,
				Threshold: genai.HarmBlockThresholdOff,
			},
			{
				Category:  genai.HarmCategoryHarassment,
				Threshold: genai.HarmBlockThresholdOff,
			},
		}
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
		genConfig,
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
	var textResponse string

	for _, part := range result.Candidates[0].Content.Parts {
		if part.Text != "" {
			textResponse = part.Text
		} else if part.InlineData != nil {
			randomName := utils.GenerateID(16)
			imageBytes := part.InlineData.Data

			// Determine file extension based on MIME type
			ext := "png"
			if part.InlineData.MIMEType == "image/jpeg" {
				ext = "jpg"
			} else if part.InlineData.MIMEType == "image/webp" {
				ext = "webp"
			}

			outputFilename := fmt.Sprintf("%s/%s.%s", destination_dir, randomName, ext)
			if err := os.WriteFile(outputFilename, imageBytes, 0644); err != nil {
				return nil, fmt.Errorf("failed to write image file: %w", err)
			}
			path = outputFilename
		}
	}

	response := &models.AIImageResponse{
		FilePath: path,
	}

	// Include text response if available (some models return both text and image)
	if textResponse != "" && path == "" {
		return nil, fmt.Errorf("no image generated, model returned text only: %s", textResponse)
	}

	return response, nil
}

// GenImageVertexAI generates an image using Vertex AI backend with project and location
func GenImageVertexAI(prompt, model, destination_dir, projectID, location string, opts ...ImageGenOption) (*models.AIImageResponse, error) {
	allOpts := append([]ImageGenOption{WithVertexAI(projectID, location)}, opts...)
	return GenImageWithConfig(prompt, model, destination_dir, allOpts...)
}
