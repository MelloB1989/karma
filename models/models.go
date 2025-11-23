package models

import "time"

type ErrorMessage struct {
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
	ErrorMsg    string `json:"error_msg"`
	UserMsg     string `json:"user_msg"`
	ErrorLevel  string `json:"error_level"`
}

type EmailBody struct {
	Text string `json:"text"`
	HTML string `json:"html"`
}

type Email struct {
	Subject string    `json:"subject"`
	Body    EmailBody `json:"body"`
}

type SingleEmailRequest struct {
	Email
	To string `json:"to"`
}

type AIRoles string

const (
	User      AIRoles = "user"
	Assistant AIRoles = "assistant"
	System    AIRoles = "system"
	Tool      AIRoles = "tool"
	Function  AIRoles = "function"
)

type AIMessage struct {
	Images     []string         `json:"images"`          //Image URLs or Base64 image data URLs
	Files      []string         `json:"files"`           //File URLs or Base64 file data URLs
	ToolCalls  []OpenAIToolCall `json:"tools,omitempty"` // Tool calls based on OpenAI standards
	ToolCallId string           `json:"tool_call_id,omitempty"`
	Message    string           `json:"message"`
	Metadata   string           `json:"metadata"` //Store metadata related to the message, you can also store stringified JSON data
	Role       AIRoles          `json:"role"`
	Timestamp  time.Time        `json:"timestamp"`
	UniqueId   string           `json:"unique_id"`
}

type OpenAIToolCall struct {
	Index    *int   `json:"index,omitempty"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type OpenAIFunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
	Strict      bool   `json:"strict,omitempty"`
}

type AIChatHistory struct {
	Messages    []AIMessage `json:"messages"`
	ChatId      string      `json:"chat_id"`
	CreatedAt   time.Time   `json:"created_at"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	SystemMsg   string      `json:"system_msg"`
	Context     string      `json:"context"`
}

type ToolCall struct {
	Index    *int             `json:"index,omitempty"`
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type AIChatResponse struct {
	AIResponse   string     `json:"ai_response"`
	Tokens       int        `json:"tokens"`
	InputTokens  int        `json:"input_tokens"`
	OutputTokens int        `json:"output_tokens"`
	TimeTaken    int        `json:"time_taken"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
}

type AIImageResponse struct {
	ImageHostedUrl string `json:"image_hosted_url"`
	FilePath       string `json:"file_path"`
}

type AIEmbeddingResponse struct {
	Embeddings []float64 `json:"embeddings"`
	Usage      struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// GetEmbeddingsFloat32 converts the embeddings to float32
func (e *AIEmbeddingResponse) GetEmbeddingsFloat32() []float32 {
	out := make([]float32, len(e.Embeddings))
	for i, v := range e.Embeddings {
		out[i] = float32(v)
	}
	return out
}

type StreamedResponse struct {
	AIResponse string     `json:"text"`
	TokenUsed  int        `json:"token_used"`
	TimeTaken  int        `json:"time_taken"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type GoogleConfig struct {
	CookieExpiration time.Duration
	CookieDomain     string
	CookieHTTPSOnly  bool
	OAuthStateString string
	UseJWT           bool
	GetClaims        func(user *GoogleCallbackData) map[string]interface{}
}

type GoogleCallbackData struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
	ID            string `json:"id"`
}
