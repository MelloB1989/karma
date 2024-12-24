package models

type GoogleCallbackData struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
	ID            string `json:"id"`
}

type ErrorMessage struct {
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
	ErrorMsg    string `json:"error_msg"`
	UserMsg     string `json:"user_msg"`
	ErrorLevel  string `json:"error_level"`
}
