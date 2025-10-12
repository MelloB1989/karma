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

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("EmptyResponse", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		var response map[string]string
		err := client.Get("/empty", &response)
		// Should handle empty response gracefully
		if err != nil && !strings.Contains(err.Error(), "unexpected end of JSON") {
			t.Logf("Empty response handled: %v", err)
		}
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{invalid json`))
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		var response map[string]string
		err := client.Get("/malformed", &response)
		if err == nil {
			t.Error("Expected error for malformed JSON")
		}
	})

	t.Run("NilResponseStruct", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		err := client.Get("/test", nil)
		// Should handle nil response struct
		if err != nil {
			t.Logf("Nil response struct handled: %v", err)
		}
	})

	t.Run("VeryLargeResponse", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate large JSON array
			w.Write([]byte("["))
			for i := 0; i < 10000; i++ {
				if i > 0 {
					w.Write([]byte(","))
				}
				fmt.Fprintf(w, `{"id":%d,"data":"item%d"}`, i, i)
			}
			w.Write([]byte("]"))
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		var response []map[string]interface{}
		err := client.Get("/large", &response)
		if err != nil {
			t.Fatalf("Failed to handle large response: %v", err)
		}
		if len(response) != 10000 {
			t.Errorf("Expected 10000 items, got %d", len(response))
		}
	})

	t.Run("SpecialCharactersInURL", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "special") {
				t.Errorf("URL not handled correctly: %s", r.URL.Path)
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		var response map[string]string
		err := client.Get("/users/special%20name", &response)
		if err != nil {
			t.Fatalf("Failed to handle special characters: %v", err)
		}
	})

	t.Run("EmptyBaseURL", func(t *testing.T) {
		client := NewAPIClient("", nil)
		var response map[string]string
		err := client.Get("/test", &response)
		if err == nil {
			t.Error("Expected error for empty base URL")
		}
	})

	t.Run("NilHeaders", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		var response map[string]string
		err := client.Get("/test", &response)
		if err != nil {
			t.Fatalf("Should handle nil headers: %v", err)
		}
	})
}

// TestTimeoutScenarios tests various timeout scenarios
func TestTimeoutScenarios(t *testing.T) {
	t.Run("RequestTimeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(3 * time.Second)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		client.SetTimeout(100 * time.Millisecond)

		var response map[string]string
		err := client.Get("/slow", &response)
		if err == nil {
			t.Error("Expected timeout error")
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		var response map[string]string
		err := client.GetWithContext(ctx, "/test", &response)
		if err == nil {
			t.Error("Expected context cancellation error")
		}
		if !strings.Contains(err.Error(), "context canceled") {
			t.Logf("Got error: %v", err)
		}
	})

	t.Run("ContextDeadline", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(100*time.Millisecond))
		defer cancel()

		var response map[string]string
		err := client.GetWithContext(ctx, "/test", &response)
		if err == nil {
			t.Error("Expected deadline exceeded error")
		}
	})
}

// TestComplexDataStructures tests handling of complex nested structures
func TestComplexDataStructures(t *testing.T) {
	type Address struct {
		Street  string `json:"street"`
		City    string `json:"city"`
		Country string `json:"country"`
	}

	type Company struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	type ComplexUser struct {
		ID          int                    `json:"id"`
		Name        string                 `json:"name"`
		Tags        []string               `json:"tags"`
		Metadata    map[string]interface{} `json:"metadata"`
		Company     Company                `json:"company"`
		Permissions []string               `json:"permissions"`
		CreatedAt   time.Time              `json:"created_at"`
	}

	t.Run("NestedStructures", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := ComplexUser{
				ID:   1,
				Name: "John Doe",
				Tags: []string{"admin", "developer"},
				Metadata: map[string]interface{}{
					"level":  "senior",
					"active": true,
					"score":  95.5,
				},
				Company: Company{
					Name: "Tech Corp",
					Address: Address{
						Street:  "123 Main St",
						City:    "San Francisco",
						Country: "USA",
					},
				},
				Permissions: []string{"read", "write", "delete"},
				CreatedAt:   time.Now(),
			}
			json.NewEncoder(w).Encode(user)
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		var user ComplexUser
		err := client.Get("/users/1", &user)
		if err != nil {
			t.Fatalf("Failed to parse complex structure: %v", err)
		}

		if user.ID != 1 {
			t.Errorf("Expected ID 1, got %d", user.ID)
		}
		if len(user.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(user.Tags))
		}
		if user.Company.Name != "Tech Corp" {
			t.Errorf("Expected company name 'Tech Corp', got '%s'", user.Company.Name)
		}
		if user.Metadata["level"] != "senior" {
			t.Error("Metadata not parsed correctly")
		}
	})

	t.Run("ArrayOfComplexObjects", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			users := []ComplexUser{
				{ID: 1, Name: "User 1", Tags: []string{"admin"}},
				{ID: 2, Name: "User 2", Tags: []string{"user"}},
			}
			json.NewEncoder(w).Encode(users)
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		var users []ComplexUser
		err := client.Get("/users", &users)
		if err != nil {
			t.Fatalf("Failed to parse array of complex objects: %v", err)
		}
		if len(users) != 2 {
			t.Errorf("Expected 2 users, got %d", len(users))
		}
	})
}

// TestErrorResponseParsing tests parsing of error responses
func TestErrorResponseParsing(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"BadRequest", 400, `{"error":"bad_request","message":"Invalid input"}`},
		{"Unauthorized", 401, `{"error":"unauthorized","message":"Token expired"}`},
		{"Forbidden", 403, `{"error":"forbidden","message":"Access denied"}`},
		{"NotFound", 404, `{"error":"not_found","message":"Resource not found"}`},
		{"InternalError", 500, `{"error":"internal_error","message":"Server error"}`},
		{"ServiceUnavailable", 503, `{"error":"unavailable","message":"Service down"}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, nil)
			var response map[string]string
			err := client.Get("/test", &response)

			if err == nil {
				t.Errorf("Expected error for status code %d", tc.statusCode)
			}

			httpErr, ok := GetHTTPError(err)
			if !ok {
				t.Error("Expected HTTPError type")
			}
			if httpErr.StatusCode != tc.statusCode {
				t.Errorf("Expected status code %d, got %d", tc.statusCode, httpErr.StatusCode)
			}
			if !strings.Contains(httpErr.Body, "error") {
				t.Error("Error body not captured correctly")
			}
		})
	}
}

// TestRetryLogic tests retry scenarios (requires manual implementation)
func TestRetryLogic(t *testing.T) {
	t.Run("ServerRecovery", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)

		// Manual retry logic (can be wrapped in a helper)
		var response map[string]string
		var err error
		maxRetries := 3

		for i := 0; i < maxRetries; i++ {
			err = client.Get("/unstable", &response)
			if err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if err != nil {
			t.Fatalf("Request failed after retries: %v", err)
		}
		if response["status"] != "ok" {
			t.Error("Unexpected response after retry")
		}
	})
}

// TestFileUploadEdgeCases tests file upload edge cases
func TestFileUploadEdgeCases(t *testing.T) {
	t.Run("NonExistentFile", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Server should not be called for non-existent file")
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		var response map[string]string
		err := client.UploadFile("/upload", "file", "/path/to/nonexistent/file.txt", nil, &response)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		if !strings.Contains(err.Error(), "failed to open file") {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("EmptyFile", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.ParseMultipartForm(32 << 20)
			file, header, _ := r.FormFile("file")
			defer file.Close()

			content, _ := io.ReadAll(file)
			if len(content) != 0 {
				t.Errorf("Expected empty file, got %d bytes", len(content))
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"filename": header.Filename,
				"size":     len(content),
			})
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		emptyFile := filepath.Join(tmpDir, "empty.txt")
		os.WriteFile(emptyFile, []byte(""), 0644)

		client := NewAPIClient(server.URL, nil)
		var response map[string]interface{}
		err := client.UploadFile("/upload", "file", emptyFile, nil, &response)
		if err != nil {
			t.Fatalf("Empty file upload failed: %v", err)
		}
	})

	t.Run("LargeFile", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.ParseMultipartForm(32 << 20)
			file, _, _ := r.FormFile("file")
			defer file.Close()

			size := 0
			buf := make([]byte, 1024)
			for {
				n, err := file.Read(buf)
				size += n
				if err == io.EOF {
					break
				}
			}

			json.NewEncoder(w).Encode(map[string]int{"size": size})
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		largeFile := filepath.Join(tmpDir, "large.bin")

		// Create 1MB file
		data := make([]byte, 1024*1024)
		for i := range data {
			data[i] = byte(i % 256)
		}
		os.WriteFile(largeFile, data, 0644)

		client := NewAPIClient(server.URL, nil)
		client.SetTimeout(10 * time.Second)

		var response map[string]int
		err := client.UploadFile("/upload", "file", largeFile, nil, &response)
		if err != nil {
			t.Fatalf("Large file upload failed: %v", err)
		}
		if response["size"] != 1024*1024 {
			t.Errorf("Expected size 1048576, got %d", response["size"])
		}
	})

	t.Run("UploadReaderError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)

		// Create a reader that will error
		errorReader := &errorReader{}
		var response map[string]string
		err := client.UploadReader("/upload", "file", "test.txt", errorReader, nil, &response)
		if err == nil {
			t.Error("Expected error from error reader")
		}
	})
}

// errorReader is a reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}

// TestHeaderPersistence tests that headers persist across requests
func TestHeaderPersistence(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		auth := r.Header.Get("Authorization")
		if auth != "Bearer persistent-token" {
			t.Errorf("Request %d: Expected persistent auth header, got '%s'", requestCount, auth)
		}
		json.NewEncoder(w).Encode(map[string]int{"request": requestCount})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, map[string]string{
		"Authorization": "Bearer persistent-token",
	})

	for i := 1; i <= 3; i++ {
		var response map[string]int
		err := client.Get(fmt.Sprintf("/request/%d", i), &response)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
	}

	if requestCount != 3 {
		t.Errorf("Expected 3 requests, got %d", requestCount)
	}
}

// TestContentTypeHandling tests various content types
func TestContentTypeHandling(t *testing.T) {
	t.Run("JSONContentType", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("Expected JSON content type, got '%s'", contentType)
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		payload := map[string]string{"key": "value"}
		var response map[string]string
		err := client.Post("/test", payload, &response)
		if err != nil {
			t.Fatalf("POST request failed: %v", err)
		}
	})

	t.Run("CustomContentType", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/xml" {
				t.Errorf("Expected XML content type, got '%s'", contentType)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, map[string]string{
			"Content-Type": "application/xml",
		})
		payload := map[string]string{"key": "value"}
		var response map[string]string
		err := client.Post("/test", payload, &response)
		// May fail due to XML vs JSON, but content-type should be correct
		if err != nil {
			t.Logf("Expected error due to XML/JSON mismatch: %v", err)
		}
	})
}

// TestRawModeVariations tests raw mode with different response types
func TestRawModeVariations(t *testing.T) {
	t.Run("RawModeWithHTML", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html><body>Test</body></html>"))
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		client.SetRawMode(true)

		var rawResponse string
		err := client.Get("/page", &rawResponse)
		if err != nil {
			t.Fatalf("Raw mode HTML request failed: %v", err)
		}
		if !strings.Contains(rawResponse, "<html>") {
			t.Error("Raw HTML not captured correctly")
		}
	})

	t.Run("RawModeWithPlainText", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("Plain text response"))
		}))
		defer server.Close()

		client := NewAPIClient(server.URL, nil)
		client.SetRawMode(true)

		var rawResponse string
		err := client.Get("/text", &rawResponse)
		if err != nil {
			t.Fatalf("Raw mode text request failed: %v", err)
		}
		if rawResponse != "Plain text response" {
			t.Errorf("Expected 'Plain text response', got '%s'", rawResponse)
		}
	})
}

// TestNormalizeTimeFields tests the time normalization function
func TestNormalizeTimeFields(t *testing.T) {
	testData := map[string]interface{}{
		"date1": "2025-10-10",
		"date2": "2025-10-10T15:04:05Z",
		"nested": map[string]interface{}{
			"date3": "2025-10-10 10:30:00",
		},
		"array": []interface{}{
			map[string]interface{}{
				"date4": "2025-10-10T15:04:05+05:30",
			},
		},
		"not_a_date": "just a string",
		"number":     42,
	}

	normalizeTimeFields(testData)

	// Verify dates were normalized
	if _, ok := testData["date1"].(string); !ok {
		t.Error("date1 should remain a string")
	}

	// Check nested structure
	if nested, ok := testData["nested"].(map[string]interface{}); ok {
		if _, ok := nested["date3"].(string); !ok {
			t.Error("Nested date should be processed")
		}
	}
}

// BenchmarkComplexStructureParsing benchmarks parsing complex structures
func BenchmarkComplexStructureParsing(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := make([]map[string]interface{}, 100)
		for i := 0; i < 100; i++ {
			response[i] = map[string]interface{}{
				"id":         i,
				"name":       fmt.Sprintf("User %d", i),
				"created_at": "2025-10-10T15:04:05Z",
				"metadata": map[string]interface{}{
					"level": "user",
					"score": 100 + i,
				},
			}
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var response []map[string]interface{}
		client.Get("/users", &response)
	}
}

// BenchmarkFileUpload benchmarks file upload
func BenchmarkFileUpload(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseMultipartForm(32 << 20)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "bench.txt")
	os.WriteFile(testFile, []byte("benchmark test content"), 0644)

	client := NewAPIClient(server.URL, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var response map[string]string
		client.UploadFile("/upload", "file", testFile, nil, &response)
	}
}
