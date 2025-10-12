package apir

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Mock response structures
type TestUser struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TestWorkflow struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// TestNewAPIClient tests the creation of a new API client
func TestNewAPIClient(t *testing.T) {
	t.Run("WithoutDebug", func(t *testing.T) {
		client := NewAPIClient("https://api.example.com", nil)
		if client == nil {
			t.Fatal("Expected client to be created")
		}
		if client.BaseURL != "https://api.example.com" {
			t.Errorf("Expected BaseURL to be 'https://api.example.com', got '%s'", client.BaseURL)
		}
		if client.DebugMode {
			t.Error("Expected DebugMode to be false")
		}
		if client.Client.Timeout != 30*time.Second {
			t.Errorf("Expected default timeout to be 30s, got %v", client.Client.Timeout)
		}
	})

	t.Run("WithDebug", func(t *testing.T) {
		client := NewAPIClient("https://api.example.com", nil, true)
		if !client.DebugMode {
			t.Error("Expected DebugMode to be true")
		}
	})

	t.Run("WithHeaders", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer token123",
			"Content-Type":  "application/json",
		}
		client := NewAPIClient("https://api.example.com", headers)
		if len(client.Headers) != 2 {
			t.Errorf("Expected 2 headers, got %d", len(client.Headers))
		}
		if client.Headers["Authorization"] != "Bearer token123" {
			t.Error("Authorization header not set correctly")
		}
	})
}

// TestSetters tests all setter methods
func TestSetters(t *testing.T) {
	client := NewAPIClient("https://api.example.com", nil)

	t.Run("SetDebugMode", func(t *testing.T) {
		client.SetDebugMode(true)
		if !client.DebugMode {
			t.Error("SetDebugMode failed")
		}
		client.SetDebugMode(false)
		if client.DebugMode {
			t.Error("SetDebugMode failed to disable")
		}
	})

	t.Run("SetRawMode", func(t *testing.T) {
		client.SetRawMode(true)
		if !client.RawMode {
			t.Error("SetRawMode failed")
		}
		client.SetRawMode(false)
		if client.RawMode {
			t.Error("SetRawMode failed to disable")
		}
	})

	t.Run("SetTimeout", func(t *testing.T) {
		client.SetTimeout(60 * time.Second)
		if client.RequestTimeout != 60*time.Second {
			t.Error("SetTimeout failed")
		}
		if client.Client.Timeout != 60*time.Second {
			t.Error("HTTP client timeout not updated")
		}
	})

	t.Run("SetHTTPClient", func(t *testing.T) {
		customClient := &http.Client{Timeout: 10 * time.Second}
		client.SetHTTPClient(customClient)
		if client.Client.Timeout != 10*time.Second {
			t.Error("SetHTTPClient failed")
		}
	})
}

// TestHeaderManagement tests header operations
func TestHeaderManagement(t *testing.T) {
	client := NewAPIClient("https://api.example.com", nil)

	t.Run("AddHeader", func(t *testing.T) {
		client.AddHeader("X-Custom-Header", "custom-value")
		if client.Headers["X-Custom-Header"] != "custom-value" {
			t.Error("AddHeader failed")
		}
	})

	t.Run("UpdateHeader", func(t *testing.T) {
		client.AddHeader("X-Custom-Header", "updated-value")
		if client.Headers["X-Custom-Header"] != "updated-value" {
			t.Error("Header update failed")
		}
	})

	t.Run("RemoveHeader", func(t *testing.T) {
		client.RemoveHeader("X-Custom-Header")
		if _, exists := client.Headers["X-Custom-Header"]; exists {
			t.Error("RemoveHeader failed")
		}
	})

	t.Run("GetHeaders", func(t *testing.T) {
		client.AddHeader("Header1", "Value1")
		client.AddHeader("Header2", "Value2")
		headers := client.GetHeaders()
		if len(headers) != 2 {
			t.Errorf("Expected 2 headers, got %d", len(headers))
		}
		// Verify it's a copy
		headers["Header3"] = "Value3"
		if _, exists := client.Headers["Header3"]; exists {
			t.Error("GetHeaders should return a copy, not reference")
		}
	})
}

// TestGETRequest tests GET requests
func TestGETRequest(t *testing.T) {
	t.Run("SuccessfulGET", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			if r.URL.Path != "/users/1" {
				t.Errorf("Expected path /users/1, got %s", r.URL.Path)
			}

			user := TestUser{
				ID:        1,
				Name:      "John Doe",
				Email:     "john@example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			json.NewEncoder(w).Encode(user)
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		var user TestUser
		err := client.Get("/users/1", &user)
		if err != nil {
			t.Fatalf("GET request failed: %v", err)
		}
		if user.ID != 1 {
			t.Errorf("Expected user ID 1, got %d", user.ID)
		}
		if user.Name != "John Doe" {
			t.Errorf("Expected name 'John Doe', got '%s'", user.Name)
		}
	})

	t.Run("GETWithHeaders", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-token" {
				t.Errorf("Expected Authorization header, got '%s'", auth)
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		headers := map[string]string{"Authorization": "Bearer test-token"}
		client := NewAPIClient(server.URL, headers)
		var response map[string]string
		err := client.Get("/test", &response)
		if err != nil {
			t.Fatalf("GET request failed: %v", err)
		}
	})

	t.Run("GETWithContext", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		var response map[string]string
		err := client.GetWithContext(ctx, "/test", &response)
		if err == nil {
			t.Error("Expected context timeout error")
		}
	})

	t.Run("GET404Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "not_found",
				Message: "User not found",
			})
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		var user TestUser
		err := client.Get("/users/999", &user)
		if err == nil {
			t.Error("Expected error for 404 response")
		}

		httpErr, ok := GetHTTPError(err)
		if !ok {
			t.Error("Expected HTTPError type")
		}
		if httpErr.StatusCode != 404 {
			t.Errorf("Expected status code 404, got %d", httpErr.StatusCode)
		}
	})
}

// TestPOSTRequest tests POST requests
func TestPOSTRequest(t *testing.T) {
	t.Run("SuccessfulPOST", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got %s", r.Method)
			}

			var requestBody map[string]string
			json.NewDecoder(r.Body).Decode(&requestBody)

			if requestBody["name"] != "Jane Doe" {
				t.Errorf("Expected name 'Jane Doe', got '%s'", requestBody["name"])
			}

			user := TestUser{
				ID:        2,
				Name:      requestBody["name"],
				Email:     requestBody["email"],
				CreatedAt: time.Now(),
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(user)
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		payload := map[string]string{
			"name":  "Jane Doe",
			"email": "jane@example.com",
		}
		var user TestUser
		err := client.Post("/users", payload, &user)
		if err != nil {
			t.Fatalf("POST request failed: %v", err)
		}
		if user.ID != 2 {
			t.Errorf("Expected user ID 2, got %d", user.ID)
		}
	})

	t.Run("POSTWithContext", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		ctx := context.Background()
		payload := map[string]string{"test": "data"}
		var response map[string]string
		err := client.PostWithContext(ctx, "/test", payload, &response)
		if err != nil {
			t.Fatalf("POST with context failed: %v", err)
		}
	})
}

// TestPUTRequest tests PUT requests
func TestPUTRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		var requestBody map[string]string
		json.NewDecoder(r.Body).Decode(&requestBody)

		user := TestUser{
			ID:        1,
			Name:      requestBody["name"],
			UpdatedAt: time.Now(),
		}
		json.NewEncoder(w).Encode(user)
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, nil)
	payload := map[string]string{"name": "Updated Name"}
	var user TestUser
	err := client.Put("/users/1", payload, &user)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	if user.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", user.Name)
	}
}

// TestPATCHRequest tests PATCH requests
func TestPATCHRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH method, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "patched"})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, nil)
	payload := map[string]string{"email": "newemail@example.com"}
	var response map[string]string
	err := client.Patch("/users/1", payload, &response)
	if err != nil {
		t.Fatalf("PATCH request failed: %v", err)
	}
}

// TestDELETERequest tests DELETE requests
func TestDELETERequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, nil)
	err := client.Delete("/users/1", nil)
	if err != nil {
		t.Fatalf("DELETE request failed: %v", err)
	}
}

// TestOtherHTTPMethods tests OPTIONS, HEAD, etc.
func TestOtherHTTPMethods(t *testing.T) {
	t.Run("OPTIONS", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodOptions {
				t.Errorf("Expected OPTIONS method, got %s", r.Method)
			}
			w.Header().Set("Allow", "GET, POST, PUT, DELETE")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		err := client.Options("/users", nil)
		if err != nil {
			t.Fatalf("OPTIONS request failed: %v", err)
		}
	})

	t.Run("HEAD", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodHead {
				t.Errorf("Expected HEAD method, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		err := client.Head("/users", nil)
		if err != nil {
			t.Fatalf("HEAD request failed: %v", err)
		}
	})
}

// TestRawMode tests raw response mode
func TestRawMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"success","data":"raw content"}`))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, nil)
	client.SetRawMode(true)

	var rawResponse string
	err := client.Get("/test", &rawResponse)
	if err != nil {
		t.Fatalf("Raw mode GET failed: %v", err)
	}

	if !strings.Contains(rawResponse, "raw content") {
		t.Errorf("Expected raw response to contain 'raw content', got: %s", rawResponse)
	}
}

// TestFlexibleTimeParsing tests various date/time formats
func TestFlexibleTimeParsing(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"DateOnly", "2025-10-10", true},
		{"RFC3339", "2025-10-10T15:04:05Z", true},
		{"RFC3339WithTZ", "2025-10-10T15:04:05+05:30", true},
		{"SpaceSeparated", "2025-10-10 15:04:05", true},
		{"InvalidFormat", "not-a-date", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tryParseTime(tc.input)
			hasResult := result != ""
			if hasResult != tc.expected {
				t.Errorf("Expected parse result %v for input '%s', got %v", tc.expected, tc.input, hasResult)
			}
		})
	}

	// Test with actual server response
	t.Run("ServerResponseWithDateOnly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// This simulates the problematic response format
			response := `{
				"id": "test123",
				"name": "Test Workflow",
				"status": "active",
				"start_date": "2025-10-10",
				"end_date": "2025-10-15"
			}`
			w.Write([]byte(response))
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		var workflow TestWorkflow
		err := client.Get("/workflow/test123", &workflow)
		if err != nil {
			t.Fatalf("Failed to parse workflow with date-only format: %v", err)
		}

		if workflow.ID != "test123" {
			t.Errorf("Expected ID 'test123', got '%s'", workflow.ID)
		}

		// Verify dates were parsed
		if workflow.StartDate.IsZero() {
			t.Error("StartDate should not be zero")
		}
		if workflow.EndDate.IsZero() {
			t.Error("EndDate should not be zero")
		}
	})
}

// TestFileUpload tests file upload functionality
func TestFileUpload(t *testing.T) {
	t.Run("SingleFileUpload", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got %s", r.Method)
			}

			err := r.ParseMultipartForm(32 << 20)
			if err != nil {
				t.Fatalf("Failed to parse multipart form: %v", err)
			}

			file, header, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("Failed to get form file: %v", err)
			}
			defer file.Close()

			if header.Filename != "test.txt" {
				t.Errorf("Expected filename 'test.txt', got '%s'", header.Filename)
			}

			content, _ := io.ReadAll(file)
			if string(content) != "test content" {
				t.Errorf("Expected content 'test content', got '%s'", string(content))
			}

			json.NewEncoder(w).Encode(map[string]string{
				"status":   "uploaded",
				"filename": header.Filename,
			})
		}))
		defer server.Close()

		// Create temporary test file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		client := NewAPIClient(server.URL, nil)
		var response map[string]string
		err = client.UploadFile("/upload", "file", testFile, nil, &response)
		if err != nil {
			t.Fatalf("File upload failed: %v", err)
		}

		if response["status"] != "uploaded" {
			t.Errorf("Expected status 'uploaded', got '%s'", response["status"])
		}
	})

	t.Run("FileUploadWithAdditionalFields", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.ParseMultipartForm(32 << 20)

			description := r.FormValue("description")
			if description != "Test description" {
				t.Errorf("Expected description 'Test description', got '%s'", description)
			}

			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("content"), 0644)

		client := NewAPIClient(server.URL, nil)
		additionalFields := map[string]string{
			"description": "Test description",
			"category":    "documents",
		}
		var response map[string]string
		err := client.UploadFile("/upload", "file", testFile, additionalFields, &response)
		if err != nil {
			t.Fatalf("File upload with fields failed: %v", err)
		}
	})

	t.Run("MultipleFileUpload", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.ParseMultipartForm(32 << 20)

			if len(r.MultipartForm.File) < 2 {
				t.Error("Expected at least 2 files")
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "uploaded",
				"file_count": len(r.MultipartForm.File),
			})
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "file2.txt")
		os.WriteFile(file1, []byte("content1"), 0644)
		os.WriteFile(file2, []byte("content2"), 0644)

		client := NewAPIClient(server.URL, nil)
		files := map[string]string{
			"file1": file1,
			"file2": file2,
		}
		var response map[string]interface{}
		err := client.UploadFiles("/upload", files, nil, &response)
		if err != nil {
			t.Fatalf("Multiple file upload failed: %v", err)
		}
	})

	t.Run("UploadFromReader", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.ParseMultipartForm(32 << 20)
			file, _, _ := r.FormFile("document")
			content, _ := io.ReadAll(file)

			json.NewEncoder(w).Encode(map[string]string{
				"status":  "uploaded",
				"content": string(content),
			})
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		reader := strings.NewReader("reader content")
		var response map[string]string
		err := client.UploadReader("/upload", "document", "doc.txt", reader, nil, &response)
		if err != nil {
			t.Fatalf("Upload from reader failed: %v", err)
		}

		if response["content"] != "reader content" {
			t.Errorf("Expected content 'reader content', got '%s'", response["content"])
		}
	})
}

// TestHTTPError tests error handling
func TestHTTPError(t *testing.T) {
	t.Run("HTTPErrorStructure", func(t *testing.T) {
		httpErr := &HTTPError{
			StatusCode: 404,
			Status:     "404 Not Found",
			Body:       `{"error":"not_found"}`,
		}

		errMsg := httpErr.Error()
		if !strings.Contains(errMsg, "404") {
			t.Errorf("Error message should contain status code: %s", errMsg)
		}
	})

	t.Run("IsHTTPError", func(t *testing.T) {
		httpErr := &HTTPError{StatusCode: 404}

		if !IsHTTPError(httpErr, 404) {
			t.Error("IsHTTPError should return true for matching status")
		}

		if IsHTTPError(httpErr, 500) {
			t.Error("IsHTTPError should return false for non-matching status")
		}

		regularErr := fmt.Errorf("regular error")
		if IsHTTPError(regularErr, 404) {
			t.Error("IsHTTPError should return false for non-HTTP errors")
		}
	})

	t.Run("GetHTTPError", func(t *testing.T) {
		httpErr := &HTTPError{StatusCode: 500, Body: "server error"}

		extracted, ok := GetHTTPError(httpErr)
		if !ok {
			t.Error("GetHTTPError should extract HTTPError")
		}
		if extracted.StatusCode != 500 {
			t.Errorf("Expected status code 500, got %d", extracted.StatusCode)
		}

		regularErr := fmt.Errorf("regular error")
		_, ok = GetHTTPError(regularErr)
		if ok {
			t.Error("GetHTTPError should return false for non-HTTP errors")
		}
	})
}

// TestDebugMode tests debug logging
func TestDebugMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, nil, true)
	var response map[string]string

	// This should produce debug output (visual inspection in test logs)
	err := client.Get("/test", &response)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

// TestConcurrentRequests tests thread safety
func TestConcurrentRequests(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		json.NewEncoder(w).Encode(map[string]int{"count": requestCount})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, nil)

	const numRequests = 10
	errChan := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			var response map[string]int
			errChan <- client.Get("/test", &response)
		}()
	}

	for i := 0; i < numRequests; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent request %d failed: %v", i, err)
		}
	}
}

// BenchmarkGETRequest benchmarks GET requests
func BenchmarkGETRequest(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var response map[string]string
		client.Get("/test", &response)
	}
}

// BenchmarkPOSTRequest benchmarks POST requests
func BenchmarkPOSTRequest(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, nil)
	payload := map[string]string{"key": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var response map[string]string
		client.Post("/test", payload, &response)
	}
}
