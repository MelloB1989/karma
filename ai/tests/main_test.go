package tests

import (
	"os"
	"testing"
)

// TestMain runs before all tests and can be used for setup/teardown
func TestMain(m *testing.M) {
	// Setup
	setupTestEnvironment()

	// Run tests
	code := m.Run()

	// Teardown
	teardownTestEnvironment()

	// Exit with the same code as the tests
	os.Exit(code)
}

// setupTestEnvironment sets up the test environment
func setupTestEnvironment() {
	// Set test environment variables if needed
	os.Setenv("KARMA_ENV", "test")

	// You can add more setup here like:
	// - Mock external services
	// - Initialize test databases
	// - Set up test configurations
}

// teardownTestEnvironment cleans up after tests
func teardownTestEnvironment() {
	// Clean up test environment
	os.Unsetenv("KARMA_ENV")

	// You can add more cleanup here like:
	// - Close database connections
	// - Clean up temporary files
	// - Reset global state
}

// Helper functions for tests

// MockAPIResponse creates a mock API response for testing
type MockAPIResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// CreateMockResponse creates a standardized mock response
func CreateMockResponse(statusCode int, body string) MockAPIResponse {
	return MockAPIResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers:    make(map[string]string),
	}
}

// IsTestEnvironment checks if we're running in test mode
func IsTestEnvironment() bool {
	return os.Getenv("KARMA_ENV") == "test"
}

// SkipIfNoAPIKey skips the test if API key environment variable is not set
func SkipIfNoAPIKey(t *testing.T, envVar string) {
	if os.Getenv(envVar) == "" {
		t.Skipf("Skipping test because %s environment variable is not set", envVar)
	}
}

// Helper test utilities

// AssertEqual compares two values for equality
func AssertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// AssertNotEqual compares two values for inequality
func AssertNotEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected == actual {
		t.Errorf("Expected %v to not equal %v", expected, actual)
	}
}

// AssertNil checks if a value is nil
func AssertNil(t *testing.T, value interface{}) {
	t.Helper()
	if value != nil {
		t.Errorf("Expected nil, got %v", value)
	}
}

// AssertNotNil checks if a value is not nil
func AssertNotNil(t *testing.T, value interface{}) {
	t.Helper()
	if value == nil {
		t.Error("Expected non-nil value, got nil")
	}
}

// AssertTrue checks if a value is true
func AssertTrue(t *testing.T, value bool) {
	t.Helper()
	if !value {
		t.Error("Expected true, got false")
	}
}

// AssertFalse checks if a value is false
func AssertFalse(t *testing.T, value bool) {
	t.Helper()
	if value {
		t.Error("Expected false, got true")
	}
}

// AssertContains checks if a string contains a substring
func AssertContains(t *testing.T, str, substr string) {
	t.Helper()
	if !contains(str, substr) {
		t.Errorf("Expected %q to contain %q", str, substr)
	}
}

// AssertNotContains checks if a string does not contain a substring
func AssertNotContains(t *testing.T, str, substr string) {
	t.Helper()
	if contains(str, substr) {
		t.Errorf("Expected %q to not contain %q", str, substr)
	}
}

// contains is a simple string contains check
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(substr) == 0 || stringContains(str, substr))
}

// stringContains performs a simple substring search
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Integration test helpers

// MockStreamCallback creates a mock callback for streaming tests
func MockStreamCallback(responses *[]string) func(chunk interface{}) error {
	return func(chunk interface{}) error {
		if responses != nil {
			*responses = append(*responses, "mock_chunk")
		}
		return nil
	}
}

// ErrorStreamCallback creates a callback that returns an error for testing
func ErrorStreamCallback(errorMsg string) func(chunk interface{}) error {
	return func(chunk interface{}) error {
		return &TestError{Message: errorMsg}
	}
}

// TestError is a custom error type for testing
type TestError struct {
	Message string
}

func (e *TestError) Error() string {
	return e.Message
}

// Performance test helpers

// RunBenchmark runs a function multiple times and measures performance
func RunBenchmark(b *testing.B, fn func()) {
	b.Helper()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fn()
	}
}

// RunMemoryBenchmark runs a memory benchmark
func RunMemoryBenchmark(b *testing.B, fn func()) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fn()
	}
}

// Test data generators

// GenerateTestModels returns a list of models for testing across all providers
func GenerateTestModels() map[string]interface{} {
	return map[string]interface{}{
		"openai_models": []string{
			"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo",
		},
		"anthropic_models": []string{
			"claude-3.5-sonnet-20241022", "claude-3-haiku-20240307",
		},
		"bedrock_models": []string{
			"meta.llama3-8b-instruct-v1:0", "anthropic.claude-3-5-sonnet-20241022-v2:0",
		},
		"google_models": []string{
			"gemini-2.5-flash", "gemini-1.5-pro",
		},
		"xai_models": []string{
			"grok-3", "grok-3-mini",
		},
	}
}

// GenerateTestPrompts returns various test prompts for different scenarios
func GenerateTestPrompts() []string {
	return []string{
		"Hello, how are you?",
		"Explain quantum computing in simple terms.",
		"What is the capital of France?",
		"Write a short poem about artificial intelligence.",
		"Solve this math problem: 2 + 2 = ?",
		"", // Empty prompt
		"A very long prompt that exceeds normal length to test handling of large inputs. " +
			"This prompt continues for multiple sentences to simulate real-world usage patterns. " +
			"It includes various types of content and should help test the system's robustness.",
		"Test with émojis 🚀 and spëcial chars: ñ, ü, ß, ç, æ, 世界",
		"Multi-line\nprompt\nwith\nbreaks",
	}
}

// Test configuration helpers

// GetTestConfig returns a standard test configuration
func GetTestConfig() map[string]interface{} {
	return map[string]interface{}{
		"temperature":    0.7,
		"max_tokens":     1000,
		"top_p":          0.9,
		"top_k":          50,
		"system_message": "You are a helpful AI assistant for testing purposes.",
	}
}

// GetMCPTestConfig returns MCP configuration for testing
func GetMCPTestConfig() map[string]interface{} {
	return map[string]interface{}{
		"mcp_url":    "http://localhost:8086/mcp",
		"auth_token": "test-token-123",
		"tools": []map[string]interface{}{
			{
				"friendly_name": "Test Calculator",
				"tool_name":     "calculate",
				"description":   "Perform basic arithmetic operations for testing",
				"input_schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"operation": map[string]interface{}{
							"type": "string",
							"enum": []string{"add", "subtract", "multiply", "divide"},
						},
						"a": map[string]interface{}{"type": "number"},
						"b": map[string]interface{}{"type": "number"},
					},
					"required": []string{"operation", "a", "b"},
				},
			},
		},
	}
}

// Cleanup helpers

// CleanupTempFiles removes temporary files created during tests
func CleanupTempFiles(patterns []string) {
	// Implementation would depend on specific file patterns
	// This is a placeholder for cleanup logic
}

// ResetGlobalState resets any global state that might affect tests
func ResetGlobalState() {
	// Reset any global variables or state that might interfere with tests
	// This is provider/implementation specific
}
