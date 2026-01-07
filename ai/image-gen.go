package ai

import (
	"errors"

	"github.com/MelloB1989/karma/apis/gemini"
	"github.com/MelloB1989/karma/apis/segmind"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/internal/openai"
	"github.com/MelloB1989/karma/models"
)

// Image Models

type ImageModels string

const (
	GROK_2_IMAGE       ImageModels = "grok-2-image"
	GPT_1_IMAGE        ImageModels = "gpt-image-1"
	DALL_E_3           ImageModels = "dall-e-3"
	DALL_E_2           ImageModels = "dall-e-2"
	GEMINI_NANO_BANANA ImageModels = "gemini-2.5-flash-image-preview"
	// Google/Gemini Image Models
	GEMINI_3_PRO_IMAGE ImageModels = "gemini-3-pro-image-preview"
	// Segmind Models
	SEGMIND_SD          ImageModels = "segmind-sd-3.5-large"
	SEGMIND_PROTOVIS    ImageModels = "segmind-protovis-lightning"
	SEGMIND_SAMARITAN   ImageModels = "segmind-samaritan-3d"
	SEGMIND_DREAMSHAPER ImageModels = "segmind-dreamshaper-lightning"
	SEGMIND_NANO_BANANA ImageModels = "segmind-nano-banana"
	SEGMIND_FLUX        ImageModels = "segmind-flux-schnell"
	SEGMIND_MIDJOURNEY  ImageModels = "segmind-midjourney"
	SEGMIND_SDXL        ImageModels = "segmind-sdxl-txt2img"
	SEGMIND_SD15        ImageModels = "segmind-sd15-txt2img"
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
	// Provider-specific configuration (same as KarmaAI)
	SpecialConfig map[SpecialConfig]any
	// Image generation settings
	AspectRatio          string  // e.g., "1:1", "16:9", "9:16", "4:3", "3:4"
	ImageSize            string  // e.g., "1K", "2K"
	MimeType             string  // e.g., "image/png", "image/jpeg"
	PersonGeneration     string  // e.g., "ALLOW_ALL", "BLOCK_ALL", "BLOCK_ONLY_ADULTS"
	Temperature          float32 // Temperature for generation
	DisableSafetyFilters bool    // Disable safety filters
}

type ImageGenOptions func(*KarmaImageGen)

func NewKarmaImageGen(model ImageModels, opts ...ImageGenOptions) *KarmaImageGen {
	ki := &KarmaImageGen{
		Model:         model,
		SpecialConfig: make(map[SpecialConfig]any),
	}
	for _, opt := range opts {
		opt(ki)
	}
	return ki
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

// WithImgSpecialConfig sets provider-specific configuration
func WithImgSpecialConfig(cfg map[SpecialConfig]any) ImageGenOptions {
	return func(k *KarmaImageGen) {
		k.SpecialConfig = cfg
	}
}

// WithImgAspectRatio sets the aspect ratio for generated images
func WithImgAspectRatio(aspectRatio string) ImageGenOptions {
	return func(k *KarmaImageGen) {
		k.AspectRatio = aspectRatio
	}
}

// WithImgImageSize sets the image size (e.g., "1K", "2K")
func WithImgImageSize(imageSize string) ImageGenOptions {
	return func(k *KarmaImageGen) {
		k.ImageSize = imageSize
	}
}

// WithImgMimeType sets the output MIME type (e.g., "image/png", "image/jpeg")
func WithImgMimeType(mimeType string) ImageGenOptions {
	return func(k *KarmaImageGen) {
		k.MimeType = mimeType
	}
}

// WithImgPersonGeneration sets the person generation policy
func WithImgPersonGeneration(policy string) ImageGenOptions {
	return func(k *KarmaImageGen) {
		k.PersonGeneration = policy
	}
}

// WithImgTemperature sets the temperature for image generation
func WithImgTemperature(temperature float32) ImageGenOptions {
	return func(k *KarmaImageGen) {
		k.Temperature = temperature
	}
}

// WithImgDisabledSafetyFilters disables all safety filters
func WithImgDisabledSafetyFilters() ImageGenOptions {
	return func(k *KarmaImageGen) {
		k.DisableSafetyFilters = true
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
	case GEMINI_NANO_BANANA:
		return ki.genGeminiImage(prompt, outputDir)
	case GEMINI_3_PRO_IMAGE:
		return ki.genGeminiImage(prompt, outputDir)
	case SEGMIND_SD:
		seg := segmind.NewSegmind(segmind.SegmindSDAPI, segmind.WithOutputDir(outputDir))
		url, err := seg.RequestCreateImage(ki.UserPrePrompt + " " + prompt)
		if err != nil {
			return nil, err
		}
		return &models.AIImageResponse{FilePath: *url}, nil
	case SEGMIND_PROTOVIS:
		seg := segmind.NewSegmind(segmind.SegmindProtovisAPI, segmind.WithOutputDir(outputDir))
		url, err := seg.RequestCreateImage(ki.UserPrePrompt + " " + prompt)
		if err != nil {
			return nil, err
		}
		return &models.AIImageResponse{FilePath: *url}, nil
	case SEGMIND_SAMARITAN:
		seg := segmind.NewSegmind(segmind.SegmindSamaritanAPI, segmind.WithOutputDir(outputDir))
		url, err := seg.RequestCreateImage(ki.UserPrePrompt + " " + prompt)
		if err != nil {
			return nil, err
		}
		return &models.AIImageResponse{FilePath: *url}, nil
	case SEGMIND_DREAMSHAPER:
		seg := segmind.NewSegmind(segmind.SegmindDreamshaperAPI, segmind.WithOutputDir(outputDir))
		url, err := seg.RequestCreateImage(ki.UserPrePrompt + " " + prompt)
		if err != nil {
			return nil, err
		}
		return &models.AIImageResponse{FilePath: *url}, nil
	case SEGMIND_FLUX:
		seg := segmind.NewSegmind(segmind.SegmindFluxAPI, segmind.WithOutputDir(outputDir))
		url, err := seg.RequestCreateImage(ki.UserPrePrompt + " " + prompt)
		if err != nil {
			return nil, err
		}
		return &models.AIImageResponse{FilePath: *url}, nil
	case SEGMIND_MIDJOURNEY:
		seg := segmind.NewSegmind(segmind.SegmindMidjourneyAPI, segmind.WithOutputDir(outputDir))
		url, err := seg.RequestCreateImage(ki.UserPrePrompt + " " + prompt)
		if err != nil {
			return nil, err
		}
		return &models.AIImageResponse{FilePath: *url}, nil
	case SEGMIND_SDXL:
		seg := segmind.NewSegmind(segmind.SegmindSDXLAPI, segmind.WithOutputDir(outputDir))
		url, err := seg.RequestCreateImage(ki.UserPrePrompt + " " + prompt)
		if err != nil {
			return nil, err
		}
		return &models.AIImageResponse{FilePath: *url}, nil
	case SEGMIND_SD15:
		seg := segmind.NewSegmind(segmind.SegmindSD15API, segmind.WithOutputDir(outputDir))
		url, err := seg.RequestCreateImage(ki.UserPrePrompt + " " + prompt)
		if err != nil {
			return nil, err
		}
		return &models.AIImageResponse{FilePath: *url}, nil
	case SEGMIND_NANO_BANANA:
		seg := segmind.NewSegmind(segmind.SegmindNanoBananaAPI, segmind.WithOutputDir(outputDir))
		url, err := seg.RequestCreateImageWithInputImage(ki.UserPrePrompt+" "+prompt, []string{})
		if err != nil {
			return nil, err
		}
		return &models.AIImageResponse{FilePath: *url}, nil
	}
	return nil, errors.New("unsupported model")
}

// genGeminiImage generates images using Google/Gemini with SpecialConfig support
func (ki *KarmaImageGen) genGeminiImage(prompt, outputDir string) (*models.AIImageResponse, error) {
	// Build options from configuration
	var opts []gemini.ImageGenOption

	// Check for API key first (from SpecialConfig or environment)
	if apiKey, ok := ki.SpecialConfig[GoogleAPIKey].(string); ok && apiKey != "" {
		opts = append(opts, gemini.WithAPIKey(apiKey))
	} else {
		// Check for Vertex AI configuration
		projectID := config.GetEnvRaw("GOOGLE_PROJECT_ID")
		location := config.GetEnvRaw("GOOGLE_LOCATION")

		// Override with SpecialConfig if set
		if configProjectID, ok := ki.SpecialConfig[GoogleProjectID].(string); ok && configProjectID != "" {
			projectID = configProjectID
		}
		if configLocation, ok := ki.SpecialConfig[GoogleLocation].(string); ok && configLocation != "" {
			location = configLocation
		}

		if projectID != "" && location != "" {
			opts = append(opts, gemini.WithVertexAI(projectID, location))
		}
		// If neither API key nor Vertex AI config, GenImageWithConfig will try env vars
	}

	// Add image generation settings if configured
	if ki.AspectRatio != "" {
		opts = append(opts, gemini.WithAspectRatio(ki.AspectRatio))
	}
	if ki.ImageSize != "" {
		opts = append(opts, gemini.WithImageSize(ki.ImageSize))
	}
	if ki.MimeType != "" {
		opts = append(opts, gemini.WithMimeType(ki.MimeType))
	}
	if ki.PersonGeneration != "" {
		opts = append(opts, gemini.WithPersonGeneration(ki.PersonGeneration))
	}
	if ki.Temperature > 0 {
		opts = append(opts, gemini.WithTemperatureImg(ki.Temperature))
	}
	if ki.DisableSafetyFilters {
		opts = append(opts, gemini.WithDisabledSafetyFilters())
	}

	return gemini.GenImageWithConfig(ki.UserPrePrompt+" "+prompt, string(ki.Model), outputDir, opts...)
}

// GenerateImagesWithInputImages generates images using input images (useful for models like Nano Banana)
func (ki *KarmaImageGen) GenerateImagesWithInputImages(prompt string, imageUrls []string) (*models.AIImageResponse, error) {
	// Set default output directory if not specified
	outputDir := ki.OutputDirectory
	if outputDir == "" {
		outputDir = "./images"
	}

	switch ki.Model {
	case SEGMIND_NANO_BANANA:
		seg := segmind.NewSegmind(segmind.SegmindNanoBananaAPI, segmind.WithOutputDir(outputDir))
		url, err := seg.RequestCreateImageWithInputImage(ki.UserPrePrompt+" "+prompt, imageUrls)
		if err != nil {
			return nil, err
		}
		return &models.AIImageResponse{FilePath: *url}, nil
	default:
		// For models that don't support input images, fall back to regular generation
		return ki.GenerateImages(prompt)
	}
}
