package parser

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/models"
	"github.com/openai/openai-go" // Import for OpenAI chunk type
)

// Parser represents the main parser configuration
type Parser struct {
	model      ai.Models
	options    []ai.Option
	client     *ai.KarmaAI
	maxRetries int
	Debug      bool
}

// ParserOption defines functional options for Parser
type ParserOption func(*Parser)

// WithMaxRetries sets the maximum number of retries for parsing
func WithMaxRetries(retries int) ParserOption {
	return func(p *Parser) {
		p.maxRetries = retries
	}
}

// WithAIClient directly sets the KarmaAI client
func WithAIClient(client *ai.KarmaAI) ParserOption {
	return func(p *Parser) {
		p.client = client
	}
}

// WithModel sets the AI model to use
func WithModel(model ai.Models) ParserOption {
	return func(p *Parser) {
		p.model = model
	}
}

// WithAIOptions sets additional options for the KarmaAI client
func WithAIOptions(options ...ai.Option) ParserOption {
	return func(p *Parser) {
		p.options = options
	}
}

func WithDebug(debug bool) ParserOption {
	return func(p *Parser) {
		p.Debug = debug
	}
}

// NewParser creates a new parser instance
func NewParser(opts ...ParserOption) *Parser {
	// Default configuration
	p := &Parser{
		model:      (ai.ApacClaude3_5Sonnet20240620V1),
		maxRetries: 3,
		Debug:      false,
	}

	// Apply options
	for _, opt := range opts {
		opt(p)
	}

	// Initialize client if not provided
	if p.client == nil {
		p.client = ai.NewKarmaAI(p.model, p.options...)
	}

	return p
}

// createPromptForStruct generates a prompt that instructs the AI to output according to the given struct
func createPromptForStruct(structType reflect.Type, prompt string, context string) string {
	// Start building the full prompt
	fullPrompt := "I need a response in the following JSON structure:\n\n"

	// Add schema description
	schemaDesc := generateSchemaDescription(structType, 0)
	fullPrompt += schemaDesc + "\n\n"

	// Add instructions for JSON output
	fullPrompt += "OUTPUT INSTRUCTIONS:\n"
	fullPrompt += "1. Respond ONLY with valid JSON matching the schema above\n"
	fullPrompt += "2. Do not include markdown code blocks, explanations, or other text\n"
	fullPrompt += "3. Ensure all required fields are filled\n"
	fullPrompt += "4. Use proper JSON formatting with double quotes for keys and string values\n\n"

	// Add context if provided
	if context != "" {
		fullPrompt += "CONTEXT:\n" + context + "\n\n"
	}

	// Add the user's original prompt
	fullPrompt += "PROMPT:\n" + prompt

	return fullPrompt
}

// generateSchemaDescription builds a description of the struct schema
func generateSchemaDescription(t reflect.Type, indent int) string {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		return generateSchemaDescription(t.Elem(), indent)
	}

	// We only handle structs
	if t.Kind() != reflect.Struct {
		return getBasicTypeDescription(t)
	}

	// Build struct description
	var sb strings.Builder
	sb.WriteString("{\n")

	numFields := t.NumField()
	for i := range make([]int, numFields) {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON field name
		jsonTag := field.Tag.Get("json")
		parts := strings.Split(jsonTag, ",")
		fieldName := parts[0]
		if fieldName == "" || fieldName == "-" {
			fieldName = field.Name
		}

		// Check if field is required
		required := true

		if slices.Contains(parts[1:], "omitempty") {
			required = false
		}

		// Add indentation
		sb.WriteString(strings.Repeat(" ", indent+2))

		// Add field name and description
		sb.WriteString(fmt.Sprintf("\"%s\": ", fieldName))

		// Add field type description
		fieldType := field.Type
		if fieldType.Kind() == reflect.Struct {
			sb.WriteString(generateSchemaDescription(fieldType, indent+2))
		} else if fieldType.Kind() == reflect.Slice {
			// For slices, describe the element type
			sb.WriteString("[")
			if fieldType.Elem().Kind() == reflect.Struct {
				sb.WriteString(generateSchemaDescription(fieldType.Elem(), indent+4))
			} else {
				sb.WriteString(getBasicTypeDescription(fieldType.Elem()))
			}
			sb.WriteString("]")
		} else if fieldType.Kind() == reflect.Map {
			// For maps, describe the value type
			sb.WriteString("{ key: ")
			sb.WriteString(getBasicTypeDescription(fieldType.Elem()))
			sb.WriteString(" }")
		} else {
			sb.WriteString(getBasicTypeDescription(fieldType))
		}

		// Add required note if needed
		if required {
			sb.WriteString(" (required)")
		}

		// Add field description from struct tag if available
		desc := field.Tag.Get("description")
		if desc != "" {
			sb.WriteString(fmt.Sprintf(" // %s", desc))
		}

		// Add comma if not the last field
		if i < t.NumField()-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	// Close struct
	sb.WriteString(strings.Repeat(" ", indent))
	sb.WriteString("}")

	return sb.String()
}

// getBasicTypeDescription returns a string representation of the type
func getBasicTypeDescription(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "number (integer)"
	case reflect.Float32, reflect.Float64:
		return "number (float)"
	case reflect.Bool:
		return "boolean"
	case reflect.Interface:
		return "any"
	case reflect.Map:
		return "object"
	case reflect.Slice, reflect.Array:
		elemType := getBasicTypeDescription(t.Elem())
		return fmt.Sprintf("array of %s", elemType)
	case reflect.Ptr:
		return getBasicTypeDescription(t.Elem())
	default:
		return t.String()
	}
}

// Parse sends a prompt to the AI and parses the response into the provided struct
func (p *Parser) Parse(prompt string, context string, output any) (time.Duration, int, error) {
	start := time.Now()
	outputType := reflect.TypeOf(output)

	// Make sure output is a pointer to a struct
	if outputType.Kind() != reflect.Ptr || outputType.Elem().Kind() != reflect.Struct {
		return time.Since(start), 0, fmt.Errorf("output must be a pointer to a struct")
	}

	// Create a prompt that instructs the AI about the expected structure
	structPrompt := createPromptForStruct(outputType.Elem(), prompt, context)

	if p.Debug {
		log.Println("Prompt: ", structPrompt)
	}

	var lastErr error
	resp, err := p.client.GenerateFromSinglePrompt(structPrompt)
	// for err != nil && p.maxRetries > 0 {
	// 	if p.Debug {
	// 		log.Println("Failed to get response, Retrying...")
	// 	}
	// 	lastErr = fmt.Errorf("AI request failed: %w", err)
	// 	resp, err = p.client.GenerateFromSinglePrompt(structPrompt)
	// 	p.maxRetries--
	// }
	promptFinal := structPrompt
	for p.maxRetries > 0 {
		resp, err = p.client.GenerateFromSinglePrompt(promptFinal)
		if err != nil {
			if p.Debug {
				log.Println("Failed to get response, Retrying...")
			}
			continue
		}
		cleanedJSON := cleanResponse(resp.AIResponse)
		err = json.Unmarshal([]byte(cleanedJSON), output)
		if err == nil {
			return time.Since(start), resp.Tokens, nil
		}
		lastErr = fmt.Errorf("JSON parsing failed: %w, Response: %s", err, cleanedJSON)
		if p.Debug {
			log.Println("Failed to parse, Retrying...")
		}
		promptFinal = fmt.Sprintf("Clean the following JSON. Error: %v\n\n"+"Please provide a response in STRICTLY valid JSON format, with NO additional text:\n\n%s", err, resp.AIResponse)
		p.maxRetries--
	}
	// for range p.maxRetries {
	// 	// Send prompt to the AI
	// 	resp, err := p.client.GenerateFromSinglePrompt(structPrompt)
	// 	if err != nil {
	// 		if p.Debug {
	// 			log.Println("Failed to get response, Retrying...")
	// 		}
	// 		lastErr = fmt.Errorf("AI request failed: %w", err)
	// 		continue
	// 	}

	// 	// Clean the response to extract just the JSON
	// 	cleanedJSON := cleanResponse(resp.AIResponse)

	// 	// Try to parse the JSON
	// 	err = json.Unmarshal([]byte(cleanedJSON), output)
	// 	if err == nil {
	// 		// Success!
	// 		return time.Since(start), resp.Tokens, nil
	// 	}

	// 	lastErr = fmt.Errorf("JSON parsing failed: %w, Response: %s", err, cleanedJSON)
	// 	if p.Debug {
	// 		log.Println("Failed to parse, Retrying...")
	// 	}

	// 	// For retries, add more explicit instructions about the failure
	// 	structPrompt = fmt.Sprintf(
	// 		"Your previous response could not be parsed correctly. Error: %v\n\n"+
	// 			"Please provide a response in STRICTLY valid JSON format, with NO additional text:\n\n%s",
	// 		err, structPrompt)
	// }

	return time.Since(start), 0, lastErr
}

// ParseChatResponse sends multiple messages and parses the final response
func (p *Parser) ParseChatResponse(messages []models.AIMessage, output any) error {
	outputType := reflect.TypeOf(output)

	// Make sure output is a pointer to a struct
	if outputType.Kind() != reflect.Ptr || outputType.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("output must be a pointer to a struct")
	}

	// Create a prompt that instructs the AI about the expected structure
	structDesc := generateSchemaDescription(outputType.Elem(), 0)

	// Add the schema instructions to the last user message
	lastUserMsgIdx := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == models.User {
			lastUserMsgIdx = i
			break
		}
	}

	if lastUserMsgIdx == -1 {
		return fmt.Errorf("no user message found in chat history")
	}

	originalMessage := messages[lastUserMsgIdx].Message
	messages[lastUserMsgIdx].Message = fmt.Sprintf(
		"%s\n\nPlease respond ONLY with valid JSON matching this schema:\n%s\n"+
			"Do not include markdown code blocks or any other text.",
		originalMessage, structDesc)

	var lastErr error
	for range p.maxRetries {
		// Send messages to the AI
		resp, err := p.client.ChatCompletion(models.AIChatHistory{Messages: messages})
		if err != nil {
			lastErr = fmt.Errorf("AI chat request failed: %w", err)
			continue
		}

		// Clean the response to extract just the JSON
		cleanedJSON := cleanResponse(resp.AIResponse)

		// Try to parse the JSON
		err = json.Unmarshal([]byte(cleanedJSON), output)
		if err == nil {
			// Success!
			return nil
		}

		lastErr = fmt.Errorf("JSON parsing failed: %w, Response: %s", err, cleanedJSON)

		// For retries, add a new message explaining the failure
		errorMsg := models.AIMessage{
			Role:    models.User,
			Message: fmt.Sprintf("Your previous response could not be parsed correctly. Error: %v\nPlease provide STRICTLY valid JSON with NO additional text matching the schema I provided.", err),
		}
		messages = append(messages, errorMsg)
	}

	return lastErr
}

// cleanResponse extracts JSON from an AI response, handling various formats
func cleanResponse(response string) string {
	// Try to find JSON between code blocks
	re := regexp.MustCompile("```(?:json)?\n?(.*?)```")
	matches := re.FindAllStringSubmatch(response, -1)
	if len(matches) > 0 {
		// Use the last match (in case there are multiple code blocks)
		return strings.TrimSpace(matches[len(matches)-1][1])
	}

	// Try to find JSON-like content (starting with { and ending with })
	re = regexp.MustCompile(`(?s)\{.*\}`)
	match := re.FindString(response)
	if match != "" {
		return strings.TrimSpace(match)
	}

	// If all else fails, return the cleaned response
	return strings.TrimSpace(response)
}

// ParseStream handles streaming responses and accumulates them before parsing
func (p *Parser) ParseStream(prompt string, context string, output any, progressCallback func(string)) error {
	outputType := reflect.TypeOf(output)

	// Make sure output is a pointer to a struct
	if outputType.Kind() != reflect.Ptr || outputType.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("output must be a pointer to a struct")
	}

	// Create a prompt that instructs the AI about the expected structure
	structPrompt := createPromptForStruct(outputType.Elem(), prompt, context)

	var fullResponse strings.Builder

	// Define the chunk handler for OpenAI-compatible ChatCompletionChunk
	chunkHandler := func(chunk openai.ChatCompletionChunk) {
		// Extract content from the chunk
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			fullResponse.WriteString(content)
			if progressCallback != nil {
				progressCallback(content)
			}
		}
	}

	var lastErr error
	for range p.maxRetries {
		// Reset the accumulated response
		fullResponse.Reset()

		// Create chat history with a single message
		chatHistory := models.AIChatHistory{
			Messages: []models.AIMessage{
				{
					Role:    models.User,
					Message: structPrompt,
				},
			},
		}

		// Send the stream request
		_, err := p.client.ChatCompletionStream(chatHistory, chunkHandler)
		if err != nil {
			lastErr = fmt.Errorf("AI stream request failed: %w", err)
			continue
		}

		// Clean the complete response to extract just the JSON
		cleanedJSON := cleanResponse(fullResponse.String())

		// Try to parse the JSON
		err = json.Unmarshal([]byte(cleanedJSON), output)
		if err == nil {
			// Success!
			return nil
		}

		lastErr = fmt.Errorf("JSON parsing failed: %w, Response: %s", err, cleanedJSON)

		// For retries, add more explicit instructions about the failure
		structPrompt = fmt.Sprintf(
			"Your previous response could not be parsed correctly. Error: %v\n\n"+
				"Please provide a response in STRICTLY valid JSON format, with NO additional text:\n\n%s",
			err, structPrompt)
	}

	return lastErr
}
