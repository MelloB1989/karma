package parser

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/models"
)

type Parser struct {
	client     *ai.KarmaAI
	maxRetries int
	debug      bool
}

type ParserOption func(*Parser)

func WithMaxRetries(retries int) ParserOption {
	return func(p *Parser) { p.maxRetries = retries }
}

func WithAIClient(client *ai.KarmaAI) ParserOption {
	return func(p *Parser) { p.client = client }
}

func WithDebug(debug bool) ParserOption {
	return func(p *Parser) { p.debug = debug }
}

func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{maxRetries: 3}
	for _, opt := range opts {
		opt(p)
	}
	if p.client == nil {
		p.client = ai.NewKarmaAI(ai.GPT4oMini, ai.OpenAI)
	}
	return p
}

func (p *Parser) Parse(prompt, context string, output any) (time.Duration, int, error) {
	start := time.Now()
	t := reflect.TypeOf(output)

	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return 0, 0, fmt.Errorf("output must be pointer to struct")
	}

	fullPrompt := buildPrompt(t.Elem(), prompt, context)

	var lastErr error
	retries := p.maxRetries

	for retries > 0 {
		resp, err := p.client.GenerateFromSinglePrompt(fullPrompt)
		if err != nil {
			retries--
			lastErr = err
			continue
		}

		cleaned := cleanJSON(resp.AIResponse)
		if err = json.Unmarshal([]byte(cleaned), output); err == nil {
			return time.Since(start), resp.Tokens, nil
		}

		lastErr = fmt.Errorf("parse error: %w", err)
		fullPrompt = fmt.Sprintf("Fix this JSON (error: %v):\n%s", err, resp.AIResponse)
		retries--
	}

	return time.Since(start), 0, lastErr
}

func (p *Parser) ParseChat(messages []models.AIMessage, output any) error {
	t := reflect.TypeOf(output)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("output must be pointer to struct")
	}

	schema := buildSchema(t.Elem(), 0)
	lastIdx := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == models.User {
			lastIdx = i
			break
		}
	}
	if lastIdx == -1 {
		return fmt.Errorf("no user message found")
	}

	messages[lastIdx].Message = fmt.Sprintf("%s\n\nRespond with valid JSON:\n%s",
		messages[lastIdx].Message, schema)

	var lastErr error
	for range p.maxRetries {
		resp, err := p.client.ChatCompletion(models.AIChatHistory{Messages: messages})
		if err != nil {
			lastErr = err
			continue
		}

		cleaned := cleanJSON(resp.AIResponse)
		if err = json.Unmarshal([]byte(cleaned), output); err == nil {
			return nil
		}

		lastErr = fmt.Errorf("parse error: %w", err)
		messages = append(messages, models.AIMessage{
			Role:    models.User,
			Message: fmt.Sprintf("Invalid JSON (error: %v). Retry with valid JSON only.", err),
		})
	}
	return lastErr
}

func buildPrompt(t reflect.Type, prompt, context string) string {
	var sb strings.Builder
	if context != "" {
		sb.WriteString("CONTEXT:\n")
		sb.WriteString(context)
		sb.WriteString("\n\n")
	}
	sb.WriteString("PROMPT:\n")
	sb.WriteString(prompt)
	sb.WriteString("\n\nJSON SCHEMA:\n")
	sb.WriteString(buildSchema(t, 0))
	sb.WriteString("\n\nReturn ONLY valid JSON. No markdown, no explanation.")
	return sb.String()
}

func buildSchema(t reflect.Type, indent int) string {
	if t.Kind() == reflect.Ptr {
		return buildSchema(t.Elem(), indent)
	}
	if t.Kind() != reflect.Struct {
		return typeDesc(t)
	}

	var sb strings.Builder
	sb.WriteString("{\n")

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		parts := strings.Split(jsonTag, ",")
		name := parts[0]
		if name == "" || name == "-" {
			name = field.Name
		}

		required := !slices.Contains(parts[1:], "omitempty")
		sb.WriteString(strings.Repeat(" ", indent+2))
		sb.WriteString(fmt.Sprintf(`"%s": `, name))

		ft := field.Type
		if ft.Kind() == reflect.Struct {
			sb.WriteString(buildSchema(ft, indent+2))
		} else if ft.Kind() == reflect.Slice {
			sb.WriteString("[")
			if ft.Elem().Kind() == reflect.Struct {
				sb.WriteString(buildSchema(ft.Elem(), indent+4))
			} else {
				sb.WriteString(typeDesc(ft.Elem()))
			}
			sb.WriteString("]")
		} else {
			sb.WriteString(typeDesc(ft))
		}

		if required {
			sb.WriteString(" (required)")
		}
		if desc := field.Tag.Get("description"); desc != "" {
			sb.WriteString(fmt.Sprintf(" // %s", desc))
		}
		if i < t.NumField()-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(strings.Repeat(" ", indent))
	sb.WriteString("}")
	return sb.String()
}

func typeDesc(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "number"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Ptr:
		return typeDesc(t.Elem())
	default:
		return "any"
	}
}

func cleanJSON(s string) string {
	if re := regexp.MustCompile("```(?:json)?\n?(.*?)```"); re.MatchString(s) {
		if m := re.FindAllStringSubmatch(s, -1); len(m) > 0 {
			return strings.TrimSpace(m[len(m)-1][1])
		}
	}
	if re := regexp.MustCompile(`(?s)\{.*\}`); re.MatchString(s) {
		return strings.TrimSpace(re.FindString(s))
	}
	return strings.TrimSpace(s)
}
