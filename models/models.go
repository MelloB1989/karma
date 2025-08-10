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
	Images    []string  `json:"images"` //Image URLs or Base64 image data URLs
	Files     []string  `json:"files"`  //File URLs or Base64 file data URLs
	Message   string    `json:"message"`
	Role      AIRoles   `json:"role"`
	Timestamp time.Time `json:"timestamp"`
	UniqueId  string    `json:"unique_id"`
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

type AIChatResponse struct {
	AIResponse   string `json:"ai_response"`
	Tokens       int    `json:"tokens"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	TimeTaken    int    `json:"time_taken"`
}

type StreamedResponse struct {
	AIResponse string `json:"text"`
	TokenUsed  int    `json:"token_used"`
	TimeTaken  int    `json:"time_taken"`
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
