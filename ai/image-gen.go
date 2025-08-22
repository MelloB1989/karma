package ai

import (
	"errors"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/internal/openai"
	"github.com/MelloB1989/karma/models"
)

// Image Models

type ImageModels string

const (
	GROK_2_IMAGE ImageModels = "grok-2-image"
	GPT_1_IMAGE  ImageModels = "gpt-image-1"
	DALL_E_3     ImageModels = "dall-e-3"
	DALL_E_2     ImageModels = "dall-e-2"
	// Models to be supported in future:
	// Gemini20FlashPreviewImageGen ImageModels = "gemini-2.0-flash-preview-image-generation"
	// StableDiffusionXLV1          ImageModels = "stability.stable-diffusion-xl-v1:0"
	// TitanImageGeneratorV1        ImageModels = "amazon.titan-image-generator-v1:0"
	// TitanImageGeneratorV2        ImageModels = "amazon.titan-image-generator-v2:0"
)

type KarmaImageGen struct {
	UserPrePrompt   string // User's pre-prompt for image generation
	NegativePrompt  string // User's negative prompt for image generation
	N               int    // Number of output images
	Model           ImageModels
	OutputDirectory string
}

type ImageGenOptions func(*KarmaImageGen)

func NewKarmaImageGen(model ImageModels, opts ...ImageGenOptions) *KarmaImageGen {

	return &KarmaImageGen{
		Model: model,
	}
}

func WithNImages(n int) ImageGenOptions {
	return func(k *KarmaImageGen) {
		k.N = n
	}
}

func WithImgUserPrePrompt(prompt string) ImageGenOptions {
	return func(k *KarmaImageGen) {
		k.UserPrePrompt = prompt
	}
}

func WithImgNegativePrompt(prompt string) ImageGenOptions {
	return func(k *KarmaImageGen) {
		k.NegativePrompt = prompt
	}
}

func WithOutputDirectory(dir string) ImageGenOptions {
	return func(k *KarmaImageGen) {
		k.OutputDirectory = dir
	}
}

func (ki *KarmaImageGen) GenerateImages(prompt string) (*models.AIImageResponse, error) {
	// Set default output directory if not specified
	outputDir := ki.OutputDirectory
	if outputDir == "" {
		outputDir = "./images"
	}

	switch ki.Model {
	case GPT_1_IMAGE:
		return openai.GenImage(ki.UserPrePrompt+" "+prompt, string(ki.Model), outputDir)
	case DALL_E_2:
		return openai.GenImage(ki.UserPrePrompt+" "+prompt, string(ki.Model), outputDir)
	case DALL_E_3:
		return openai.GenImage(ki.UserPrePrompt+" "+prompt, string(ki.Model), outputDir)
	case GROK_2_IMAGE:
		return openai.GenImage(ki.UserPrePrompt+" "+prompt, string(ki.Model), outputDir, openai.CompatibleOptions{
			BaseURL: XAI_API,
			API_Key: config.GetEnvRaw("XAI_API_KEY"),
		})
	}
	return nil, errors.New("unsupported model")
}
