# Karma AI Package

A flexible and extensible AI package that supports multiple model providers with a clean, unified interface.

## Overview

The Karma AI package has been restructured to separate base models from providers, making it easy to:
- Use the same model with different providers
- Add new providers without duplicating models
- Switch between providers seamlessly
- Maintain backward compatibility

## Key Concepts

### Base Models
Base models represent the core AI model without provider-specific naming:
- `ai.GPT4`, `ai.GPT4o`, `ai.Claude35Sonnet`, `ai.Llama31_70B`, etc.

### Providers
Providers are the services that host and serve the models:
- `ai.OpenAI`, `ai.Anthropic`, `ai.Bedrock`, `ai.Google`, `ai.XAI`, `ai.Groq`, etc.

### Model Configuration
`ModelConfig` combines a base model with a provider and handles the mapping to provider-specific model strings.

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/MelloB1989/karma/ai"
    "github.com/MelloB1989/karma/ai/providers"
)

func main() {
    // Initialize provider mappings (do this once in your application)
    ai.InitializeProviderMappings(providers.ProviderModelMapping)
    
    // Create AI instance with default provider
    gpt4AI := ai.NewKarmaAI(
        ai.GPT4,    // Base model
        "",         // Empty provider = use default
        ai.WithSystemMessage("You are a helpful assistant."),
        ai.WithTemperature(0.7),
        ai.WithMaxTokens(1000),
    )
    
    // Create AI instance with specific provider
    claudeAI := ai.NewKarmaAI(
        ai.Claude35Sonnet, // Base model
        ai.Bedrock,        // Use Bedrock instead of default Anthropic
        ai.WithSystemMessage("You are a coding assistant."),
        ai.WithTemperature(0.5),
    )
    
    // Create AI instance with custom model string for specific variant
    customClaudeAI := ai.NewKarmaAIWithCustomModel(
        ai.Claude4Sonnet,           // Base model
        ai.Anthropic,               // Provider
        "claude-sonnet-4.0",        // Custom model string for specific variant
        ai.WithSystemMessage("Using specific Claude 4 variant."),
    )
}
```

### Provider Discovery

```go
// Find all providers that support a model
providers := ai.GetAvailableProviders(ai.Llama31_70B)
// Returns: [bedrock, groq, together, meta]

// Find all models available on a provider
models := providers.GetAvailableModelsForProvider(ai.Groq)
// Returns all base models supported by Groq
```

### Easy Provider Switching

```go
baseModel := ai.Llama31_70B

// Create instances with different providers for the same model
bedrockAI := ai.NewKarmaAI(baseModel, ai.Bedrock)
groqAI := ai.NewKarmaAI(baseModel, ai.Groq)
togetherAI := ai.NewKarmaAI(baseModel, ai.Together)

// Each will use the provider-specific model string:
// - Bedrock: "meta.llama3-1-70b-instruct-v1:0"
// - Groq: "llama-3.1-70b-versatile"
// - Together: "meta-llama/Llama-3.1-70B-Instruct-Turbo"

// Use custom model string for specific variants
customModelAI := ai.NewKarmaAIWithCustomModel(
    ai.Claude4Sonnet,        // Base model
    ai.Anthropic,            // Provider
    "claude-sonnet-4.0",     // Custom model string
    ai.WithSystemMessage("Using specific model variant"),
)
```

## Available Models and Providers

### OpenAI Models
- **GPT-4 family**: `GPT4`, `GPT4o`, `GPT4oMini`, `GPT4Turbo`
- **GPT-3.5**: `GPT35Turbo`
- **GPT-5 family**: `GPT5`, `GPT5Nano`, `GPT5Mini`
- **O1 family**: `O1`, `O1Mini`, `O1Preview`
- **Default Provider**: `OpenAI`

### Claude Models
- **Claude 3.5**: `Claude35Sonnet`, `Claude35Haiku`
- **Claude 3**: `Claude3Sonnet`, `Claude3Haiku`, `Claude3Opus`
- **Claude 3.7**: `Claude37Sonnet`
- **Claude 4**: `Claude4Sonnet`, `Claude4Opus`
- **Legacy**: `ClaudeInstant`, `ClaudeV2`
- **Default Provider**: `Anthropic`
- **Alternative Providers**: `Bedrock`

### Llama Models
- **Llama 3**: `Llama3_8B`, `Llama3_70B`
- **Llama 3.1**: `Llama31_8B`, `Llama31_70B`
- **Llama 3.2**: `Llama32_1B`, `Llama32_3B`, `Llama32_11B`, `Llama32_90B`
- **Llama 3.3**: `Llama33_70B`
- **Default Provider**: `Bedrock`
- **Alternative Providers**: `Groq`, `Together`, `Meta`

### Other Models
- **Mistral**: `Mistral7B`, `Mixtral8x7B`, `MistralLarge`, `MistralSmall`
- **Amazon Titan**: `TitanTextG1Large`, `TitanTextPremier`, etc.
- **Amazon Nova**: `NovaPro`, `NovaLite`, `NovaCanvas`, etc.
- **Google Gemini**: `Gemini15Flash`, `Gemini15Pro`, `Gemini20Flash`, etc.
- **xAI Grok**: `Grok3`, `Grok3Mini`, `Grok4`

## Advanced Features

### MCP (Model Context Protocol) Support

```go
// Create MCP tools
mcpTools := []ai.MCPTool{
    {
        FriendlyName: "Web Search",
        ToolName:     "web_search",
        Description:  "Search the web for information",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "query": map[string]interface{}{
                    "type": "string",
                    "description": "The search query",
                },
            },
            "required": []string{"query"},
        },
    },
}

// Create MCP server
mcpServer := ai.NewMCPServer(
    "https://api.example.com/mcp",
    "your-auth-token",
    mcpTools,
)

// Use with Claude (supports MCP)
claudeWithMCP := ai.NewKarmaAI(
    ai.Claude35Sonnet,
    ai.Anthropic,
    ai.SetMCPServers([]ai.MCPServer{mcpServer}),
)
claudeWithMCP.EnableTools()
```

### Analytics Configuration

```go
aiWithAnalytics := ai.NewKarmaAI(
    ai.GPT4o,
    ai.OpenAI,
    ai.ConfigureAnalytics(
        "user-123",         // Distinct ID
        "conversation-abc", // Trace ID
        true,               // Capture user prompts
        true,               // Capture AI responses
        true,               // Capture tool calls
    ),
)
```

### Custom Model Strings

When you need to use a specific model variant that differs from the canonical one:

```go
// Use specific Claude 4 variant
specificClaude := ai.NewKarmaAIWithCustomModel(
    ai.Claude4Sonnet,        // Base model  
    ai.Anthropic,            // Provider
    "claude-sonnet-4.0",     // Specific variant
    ai.WithSystemMessage("Using Claude Sonnet 4.0 specifically"),
)

// Use GPT-4o with specific date
specificGPT4o := ai.NewKarmaAIWithCustomModel(
    ai.GPT4o,
    ai.OpenAI, 
    "gpt-4o-2024-11-20",     // November 2024 variant
)

// Use Bedrock Claude with 200k context
claudeHighContext := ai.NewKarmaAIWithCustomModel(
    ai.Claude35Sonnet,
    ai.Bedrock,
    "anthropic.claude-3-5-sonnet-20240620-v1:0:200k", // 200k context variant
)
```

## Adding New Providers

Adding a new provider is straightforward:

1. **Add provider constant** (in `basics.go`):
```go
const NewProvider Provider = "new-provider"
```

2. **Add model mappings** (in `providers/mappings.go`):
```go
ai.NewProvider: {
    "provider-specific-model-name-1": ai.Llama3_70B,
    "provider-specific-model-name-2": ai.GPT4,
    // ... other mappings
},
```

3. **Use immediately**:
```go
ai := ai.NewKarmaAI(ai.Llama3_70B, ai.NewProvider)
```

## Migration from Old Structure

### Before (Old Structure)
```go
ai := ai.NewKarmaAI(
    ai.ChatModelGPT4o,  // Provider-specific constant
    ai.WithTemperature(0.7),
)
```

### After (New Structure)
```go
ai := ai.NewKarmaAI(
    ai.GPT4o,    // Base model
    ai.OpenAI,   // Provider
    ai.WithTemperature(0.7),
)
```

### Backward Compatibility
The old constants and methods are still available for backward compatibility:
```go
// This still works
oldModel := ai.Models("gpt-4o")
isOpenAI := oldModel.IsOpenAIModel()

// Migrating specific model variants:
// OLD: ai.NewKarmaAI(ai.ChatModelGPT4o2024_11_20, ...)  
// NEW: ai.NewKarmaAIWithCustomModel(ai.GPT4o, ai.OpenAI, "gpt-4o-2024-11-20", ...)
```

## Configuration Options

### Model Configuration
- `WithSystemMessage(string)` - Set system message
- `WithContext(string)` - Set context
- `WithUserPrePrompt(string)` - Set user pre-prompt
- `WithTemperature(float32)` - Set temperature (0.0-1.0)
- `WithMaxTokens(int)` - Set max tokens
- `WithTopP(float32)` - Set top-p value
- `WithTopK(int)` - Set top-k value
- `WithResponseType(string)` - Set response type

### Tool Configuration
- `SetMCPTools([]MCPTool)` - Set MCP tools
- `SetMCPServers([]MCPServer)` - Set MCP servers
- `AddMCPServer(MCPServer)` - Add an MCP server
- `SetMCPUrl(string)` - Set MCP URL
- `SetMCPAuthToken(string)` - Set MCP auth token

### Analytics Configuration
- `ConfigureAnalytics(distinctID, traceID, capturePrompts, captureResponses, captureToolCalls)`

## Model Methods

Each `ModelConfig` provides these methods:
- `GetModelString()` - Get provider-specific model string
- `GetProvider()` - Get the provider
- `IsOpenAIModel()` - Check if OpenAI model
- `IsOpenAICompatibleModel()` - Check if OpenAI API compatible
- `IsAnthropicModel()` - Check if Anthropic model
- `IsBedrockModel()` - Check if available on Bedrock
- `SupportsMCP()` - Check if supports MCP
- And many more...

## Examples

See the `examples/usage.go` file for comprehensive examples covering:
- Basic usage
- Provider discovery
- MCP configuration
- Analytics setup
- Provider switching
- Adding new providers
- Backward compatibility

## Benefits of New Structure

1. **Simplified Management**: No more duplicate model constants for different providers
2. **Easy Provider Addition**: Add new providers without touching existing model definitions
3. **Provider Flexibility**: Switch providers for the same model effortlessly
4. **Clean Separation**: Clear distinction between models and providers
5. **Backward Compatible**: Existing code continues to work
6. **Type Safety**: Strong typing for models and providers
7. **Discoverable**: Easy to find available models and providers
8. **Extensible**: Simple to extend with new capabilities

## When to Use Custom Model Strings

Use `NewKarmaAIWithCustomModel()` when you need:

1. **Specific model versions/dates**: `"gpt-4o-2024-11-20"` vs `"gpt-4o"`
2. **Context length variants**: `"amazon.titan-text-express-v1:0:8k"` vs default
3. **Regional variants**: `"apac.anthropic.claude-3-5-sonnet-20240620-v1:0"`
4. **Special capabilities**: `"gpt-4o-audio-preview"` for audio support
5. **Multiple variants exist**: When there are several variants of the same base model

Otherwise, use the standard `NewKarmaAI()` with canonical models.

## Model Mapping Strategy

The package uses **canonical mappings** - one preferred variant per base model per provider:

- **Claude 4 Sonnet + Anthropic** → `"claude-4-sonnet-20250514"` (canonical)
- **Claude 4 Sonnet + Anthropic** + custom → `"claude-sonnet-4.0"` (if specified)

This approach:
- ✅ Eliminates the ambiguity of multiple mappings
- ✅ Provides sensible defaults for most use cases  
- ✅ Allows precision when needed via custom strings
- ✅ Keeps the API simple for common scenarios

## Best Practices

1. **Initialize once**: Call `ai.InitializeProviderMappings()` once at application startup
2. **Use base models**: Prefer base model constants over legacy provider-specific ones
3. **Explicit providers**: Be explicit about providers for clarity
4. **Provider discovery**: Use discovery functions to find available options
5. **Custom strings for variants**: Use `NewKarmaAIWithCustomModel()` only when you need specific variants
6. **Document custom choices**: Comment why you're using a specific variant
7. **Create constants**: Define constants for commonly used custom model strings
8. **Error handling**: Check provider support before using specific features
9. **Default providers**: Rely on sensible defaults when provider doesn't matter