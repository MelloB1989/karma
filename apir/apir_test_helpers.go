package apir

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestHelper provides utility functions for testing
type TestHelper struct {
	t *testing.T
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{t: t}
}

// AssertNoError fails the test if err is not nil
func (h *TestHelper) AssertNoError(err error, message string) {
	if err != nil {
		h.t.Fatalf("%s: %v", message, err)
	}
}

// AssertError fails the test if err is nil
func (h *TestHelper) AssertError(err error, message string) {
	if err == nil {
		h.t.Fatalf("%s: expected error but got nil", message)
	}
}

// AssertEqual checks if two values are equal
func (h *TestHelper) AssertEqual(expected, actual interface{}, message string) {
	if expected != actual {
		h.t.Errorf("%s: expected %v, got %v", message, expected, actual)
	}
}

// AssertNotEqual checks if two values are not equal
func (h *TestHelper) AssertNotEqual(expected, actual interface{}, message string) {
	if expected == actual {
		h.t.Errorf("%s: expected values to be different, both are %v", message, expected)
	}
}

// AssertTrue checks if condition is true
func (h *TestHelper) AssertTrue(condition bool, message string) {
	if !condition {
		h.t.Errorf("%s: expected true, got false", message)
	}
}

// AssertFalse checks if condition is false
func (h *TestHelper) AssertFalse(condition bool, message string) {
	if condition {
		h.t.Errorf("%s: expected false, got true", message)
	}
}

// MockServer represents a configurable mock HTTP server
type MockServer struct {
	Server          *httptest.Server
	RequestCount    int
	LastRequest     *http.Request
	ResponseStatus  int
	ResponseBody    interface{}
	ResponseDelay   time.Duration
	ResponseHeaders map[string]string
	Handler         http.HandlerFunc
}

// NewMockServer creates a new mock server with default settings
func NewMockServer() *MockServer {
	mock := &MockServer{
		ResponseStatus:  http.StatusOK,
		ResponseHeaders: make(map[string]string),
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.RequestCount++
		mock.LastRequest = r

		// Add delay if configured
		if mock.ResponseDelay > 0 {
			time.Sleep(mock.ResponseDelay)
		}

		// Use custom handler if provided
		if mock.Handler != nil {
			mock.Handler(w, r)
			return
		}

		// Set custom headers
		for key, value := range mock.ResponseHeaders {
			w.Header().Set(key, value)
		}

		// Set status code
		w.WriteHeader(mock.ResponseStatus)

		// Write response body
		if mock.ResponseBody != nil {
			json.NewEncoder(w).Encode(mock.ResponseBody)
		}
	}))

	return mock
}

// SetResponse configures the mock server response
func (m *MockServer) SetResponse(status int, body interface{}) *MockServer {
	m.ResponseStatus = status
	m.ResponseBody = body
	return m
}

// SetDelay configures a response delay
func (m *MockServer) SetDelay(delay time.Duration) *MockServer {
	m.ResponseDelay = delay
	return m
}

// SetHeader adds a response header
func (m *MockServer) SetHeader(key, value string) *MockServer {
	m.ResponseHeaders[key] = value
	return m
}

// SetHandler sets a custom handler function
func (m *MockServer) SetHandler(handler http.HandlerFunc) *MockServer {
	m.Handler = handler
	return m
}

// Reset resets the mock server state
func (m *MockServer) Reset() {
	m.RequestCount = 0
	m.LastRequest = nil
	m.ResponseStatus = http.StatusOK
	m.ResponseBody = nil
	m.ResponseDelay = 0
	m.ResponseHeaders = make(map[string]string)
	m.Handler = nil
}

// Close closes the mock server
func (m *MockServer) Close() {
	m.Server.Close()
}

// URL returns the server URL
func (m *MockServer) URL() string {
	return m.Server.URL
}

// TestUsageExamples demonstrates how to use the test helpers
func TestUsageExamples(t *testing.T) {
	t.Run("UsingTestHelper", func(t *testing.T) {
		helper := NewTestHelper(t)

		client := NewAPIClient("https://api.example.com", nil)
		helper.AssertNotEqual(nil, client, "Client should not be nil")
		helper.AssertEqual("https://api.example.com", client.BaseURL, "BaseURL should match")
		helper.AssertFalse(client.DebugMode, "Debug mode should be off by default")
	})

	t.Run("UsingMockServer", func(t *testing.T) {
		mock := NewMockServer()
		defer mock.Close()

		mock.SetResponse(http.StatusOK, map[string]string{
			"message": "success",
			"status":  "ok",
		})

		client := NewAPIClient(mock.URL(), nil)
		var response map[string]string
		err := client.Get("/test", &response)

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if mock.RequestCount != 1 {
			t.Errorf("Expected 1 request, got %d", mock.RequestCount)
		}

		if response["status"] != "ok" {
			t.Errorf("Expected status 'ok', got '%s'", response["status"])
		}
	})

	t.Run("MockServerWithDelay", func(t *testing.T) {
		mock := NewMockServer()
		defer mock.Close()

		mock.SetResponse(http.StatusOK, map[string]string{"status": "ok"}).
			SetDelay(100 * time.Millisecond)

		client := NewAPIClient(mock.URL(), nil)
		start := time.Now()

		var response map[string]string
		err := client.Get("/slow", &response)

		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if elapsed < 100*time.Millisecond {
			t.Error("Expected delay was not applied")
		}
	})

	t.Run("MockServerCustomHandler", func(t *testing.T) {
		mock := NewMockServer()
		defer mock.Close()

		mock.SetHandler(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)

			response := map[string]string{
				"received": body["message"],
				"status":   "processed",
			}
			json.NewEncoder(w).Encode(response)
		})

		client := NewAPIClient(mock.URL(), nil)
		payload := map[string]string{"message": "hello"}
		var response map[string]string

		err := client.Post("/process", payload, &response)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if response["received"] != "hello" {
			t.Errorf("Expected 'hello', got '%s'", response["received"])
		}
	})
}

// TestRESTfulScenarios tests common RESTful API scenarios
func TestRESTfulScenarios(t *testing.T) {
	t.Run("CRUDOperations", func(t *testing.T) {
		// In-memory storage for mock server
		users := make(map[int]map[string]interface{})
		nextID := 1

		mock := NewMockServer()
		defer mock.Close()

		mock.SetHandler(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				// CREATE
				var user map[string]interface{}
				json.NewDecoder(r.Body).Decode(&user)
				user["id"] = nextID
				users[nextID] = user
				nextID++
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(user)

			case http.MethodGet:
				// READ
				var result []map[string]interface{}
				for _, user := range users {
					result = append(result, user)
				}
				json.NewEncoder(w).Encode(result)

			case http.MethodPut:
				// UPDATE
				var user map[string]interface{}
				json.NewDecoder(r.Body).Decode(&user)
				id := int(user["id"].(float64))
				users[id] = user
				json.NewEncoder(w).Encode(user)

			case http.MethodDelete:
				// DELETE
				// Parse ID from path (simplified)
				w.WriteHeader(http.StatusNoContent)
			}
		})

		client := NewAPIClient(mock.URL(), nil)
		helper := NewTestHelper(t)

		// CREATE
		newUser := map[string]interface{}{"name": "John Doe", "email": "john@example.com"}
		var createdUser map[string]interface{}
		err := client.Post("/users", newUser, &createdUser)
		helper.AssertNoError(err, "Create user failed")
		helper.AssertNotEqual(nil, createdUser["id"], "User ID should be set")

		// READ
		var userList []map[string]interface{}
		err = client.Get("/users", &userList)
		helper.AssertNoError(err, "Read users failed")
		helper.AssertTrue(len(userList) > 0, "Should have at least one user")

		// UPDATE
		createdUser["name"] = "Jane Doe"
		var updatedUser map[string]interface{}
		err = client.Put("/users/1", createdUser, &updatedUser)
		helper.AssertNoError(err, "Update user failed")
		helper.AssertEqual("Jane Doe", updatedUser["name"], "Name should be updated")

		// DELETE
		err = client.Delete("/users/1", nil)
		helper.AssertNoError(err, "Delete user failed")
	})

	t.Run("PaginationScenario", func(t *testing.T) {
		mock := NewMockServer()
		defer mock.Close()

		mock.SetHandler(func(w http.ResponseWriter, r *http.Request) {
			page := r.URL.Query().Get("page")
			limit := r.URL.Query().Get("limit")

			pageNum := 1
			if page != "" {
				fmt.Sscanf(page, "%d", &pageNum)
			}

			limitNum := 10
			if limit != "" {
				fmt.Sscanf(limit, "%d", &limitNum)
			}

			response := map[string]interface{}{
				"page":     pageNum,
				"limit":    limitNum,
				"total":    100,
				"data":     []map[string]string{},
				"has_next": pageNum*limitNum < 100,
			}

			json.NewEncoder(w).Encode(response)
		})

		client := NewAPIClient(mock.URL(), nil)
		var response map[string]interface{}
		err := client.Get("/users?page=2&limit=20", &response)

		if err != nil {
			t.Fatalf("Pagination request failed: %v", err)
		}

		if response["page"].(float64) != 2 {
			t.Errorf("Expected page 2, got %v", response["page"])
		}
	})

	t.Run("SearchAndFilterScenario", func(t *testing.T) {
		mock := NewMockServer()
		defer mock.Close()

		allUsers := []map[string]interface{}{
			{"id": 1, "name": "John Doe", "role": "admin"},
			{"id": 2, "name": "Jane Smith", "role": "user"},
			{"id": 3, "name": "Bob Johnson", "role": "user"},
		}

		mock.SetHandler(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query().Get("q")
			role := r.URL.Query().Get("role")

			var filtered []map[string]interface{}
			for _, user := range allUsers {
				matches := true

				if query != "" {
					name := user["name"].(string)
					if !containsIgnoreCase(name, query) {
						matches = false
					}
				}

				if role != "" && user["role"] != role {
					matches = false
				}

				if matches {
					filtered = append(filtered, user)
				}
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"results": filtered,
				"count":   len(filtered),
			})
		})

		client := NewAPIClient(mock.URL(), nil)

		// Search by name
		var searchResult map[string]interface{}
		err := client.Get("/users?q=john", &searchResult)
		if err != nil {
			t.Fatalf("Search request failed: %v", err)
		}

		// Filter by role
		var filterResult map[string]interface{}
		err = client.Get("/users?role=user", &filterResult)
		if err != nil {
			t.Fatalf("Filter request failed: %v", err)
		}
	})
}

// Helper function for case-insensitive string matching
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}

// TestAuthenticationScenarios tests various authentication patterns
func TestAuthenticationScenarios(t *testing.T) {
	t.Run("BearerTokenAuth", func(t *testing.T) {
		mock := NewMockServer()
		defer mock.Close()

		mock.SetHandler(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer valid-token-123" {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "unauthorized",
				})
				return
			}

			json.NewEncoder(w).Encode(map[string]string{
				"message": "authenticated",
			})
		})

		client := NewAPIClient(mock.URL(), map[string]string{
			"Authorization": "Bearer valid-token-123",
		})

		var response map[string]string
		err := client.Get("/protected", &response)

		if err != nil {
			t.Fatalf("Authenticated request failed: %v", err)
		}

		if response["message"] != "authenticated" {
			t.Error("Authentication failed")
		}
	})

	t.Run("APIKeyAuth", func(t *testing.T) {
		mock := NewMockServer()
		defer mock.Close()

		mock.SetHandler(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != "secret-key-456" {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "invalid_api_key",
				})
				return
			}

			json.NewEncoder(w).Encode(map[string]string{
				"status": "success",
			})
		})

		client := NewAPIClient(mock.URL(), map[string]string{
			"X-API-Key": "secret-key-456",
		})

		var response map[string]string
		err := client.Get("/api/data", &response)

		if err != nil {
			t.Fatalf("API key request failed: %v", err)
		}
	})

	t.Run("TokenRefreshScenario", func(t *testing.T) {
		mock := NewMockServer()
		defer mock.Close()

		accessToken := "initial-token"
		requestCount := 0

		mock.SetHandler(func(w http.ResponseWriter, r *http.Request) {
			requestCount++

			if r.URL.Path == "/refresh" {
				json.NewEncoder(w).Encode(map[string]string{
					"access_token": "new-token-" + fmt.Sprint(requestCount),
				})
				return
			}

			auth := r.Header.Get("Authorization")
			expectedToken := "Bearer " + accessToken

			if auth != expectedToken {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "token_expired",
				})
				return
			}

			json.NewEncoder(w).Encode(map[string]string{
				"data": "protected_data",
			})
		})

		client := NewAPIClient(mock.URL(), map[string]string{
			"Authorization": "Bearer " + accessToken,
		})

		// Simulate token refresh logic
		var response map[string]string
		err := client.Get("/protected", &response)

		if err != nil {
			// Token expired, refresh it
			var refreshResponse map[string]string
			refreshErr := client.Post("/refresh", map[string]string{
				"refresh_token": "refresh-token",
			}, &refreshResponse)

			if refreshErr != nil {
				t.Fatalf("Token refresh failed: %v", refreshErr)
			}

			// Update token
			accessToken = refreshResponse["access_token"]
			client.AddHeader("Authorization", "Bearer "+accessToken)

			// Retry original request
			err = client.Get("/protected", &response)
		}

		if err != nil {
			t.Fatalf("Request after refresh failed: %v", err)
		}
	})
}

// TestErrorHandlingPatterns tests common error handling patterns
func TestErrorHandlingPatterns(t *testing.T) {
	t.Run("ValidationErrors", func(t *testing.T) {
		mock := NewMockServer()
		defer mock.Close()

		mock.SetHandler(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)

			if body["email"] == nil || body["email"] == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "validation_error",
					"details": map[string]string{
						"email": "Email is required",
					},
				})
				return
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"status": "created"})
		})

		client := NewAPIClient(mock.URL(), nil)

		// Test with invalid data
		invalidPayload := map[string]interface{}{"name": "John"}
		var response map[string]interface{}
		err := client.Post("/users", invalidPayload, &response)

		if err == nil {
			t.Error("Expected validation error")
		}

		httpErr, ok := GetHTTPError(err)
		if !ok || httpErr.StatusCode != 400 {
			t.Error("Expected 400 Bad Request")
		}
	})

	t.Run("RateLimitError", func(t *testing.T) {
		mock := NewMockServer()
		defer mock.Close()

		requestCount := 0

		mock.SetHandler(func(w http.ResponseWriter, r *http.Request) {
			requestCount++

			if requestCount > 3 {
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprint(time.Now().Add(time.Minute).Unix()))
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "rate_limit_exceeded",
				})
				return
			}

			w.Header().Set("X-RateLimit-Remaining", fmt.Sprint(3-requestCount))
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		})

		client := NewAPIClient(mock.URL(), nil)

		for i := 1; i <= 5; i++ {
			var response map[string]string
			err := client.Get("/api", &response)

			if i <= 3 {
				if err != nil {
					t.Errorf("Request %d should succeed: %v", i, err)
				}
			} else {
				if err == nil {
					t.Errorf("Request %d should be rate limited", i)
				}
				if IsHTTPError(err, 429) {
					t.Logf("Request %d correctly rate limited", i)
				}
			}
		}
	})
}
