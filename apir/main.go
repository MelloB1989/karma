package apir

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type APIClient struct {
	BaseURL        string
	Headers        map[string]string
	DebugMode      bool
	RawMode        bool
	Client         *http.Client
	RequestTimeout time.Duration
}

func NewAPIClient(baseURL string, headers map[string]string, debug ...bool) *APIClient {
	var debugMode bool
	if len(debug) > 0 {
		debugMode = debug[0]
	}

	return &APIClient{
		BaseURL:        baseURL,
		Headers:        headers,
		DebugMode:      debugMode,
		Client:         &http.Client{Timeout: 30 * time.Second},
		RequestTimeout: 30 * time.Second,
	}
}

func (client *APIClient) SetDebugMode(debug bool) {
	client.DebugMode = debug
}

func (client *APIClient) SetRawMode(raw bool) {
	client.RawMode = raw
}

func (client *APIClient) SetTimeout(timeout time.Duration) {
	client.RequestTimeout = timeout
	client.Client.Timeout = timeout
}

func (client *APIClient) SetHTTPClient(httpClient *http.Client) {
	client.Client = httpClient
}

func (client *APIClient) sendRequest(method, endpoint string, body any) ([]byte, error) {
	return client.sendRequestWithContext(context.Background(), method, endpoint, body)
}

func (client *APIClient) sendRequestWithContext(ctx context.Context, method, endpoint string, body any) ([]byte, error) {
	url := fmt.Sprintf("%s%s", client.BaseURL, endpoint)

	var reqBody io.Reader
	var bodyBytes []byte

	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range client.Headers {
		req.Header.Set(key, value)
	}

	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	if client.DebugMode {
		log.Printf("[APIR] Request: %s %s", method, url)
		if bodyBytes != nil {
			log.Printf("[APIR] Request Body: %s", string(bodyBytes))
		}
		log.Printf("[APIR] Headers: %v", client.Headers)
	}

	resp, err := client.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if client.DebugMode {
		log.Printf("[APIR] Response Status: %d %s", resp.StatusCode, resp.Status)
		log.Printf("[APIR] Response Body: %s", string(respBody))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(respBody),
		}
	}

	return respBody, nil
}

type HTTPError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

func (client *APIClient) unmarshalResponse(respBody []byte, responseStruct any) error {
	if client.RawMode {
		if strPtr, ok := responseStruct.(*string); ok {
			*strPtr = string(respBody)
			return nil
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(respBody))
	decoder.UseNumber()

	if err := decoder.Decode(responseStruct); err != nil {

		return unmarshalWithFlexibleTime(respBody, responseStruct)
	}

	return nil
}

func unmarshalWithFlexibleTime(data []byte, v any) error {

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {

		var rawArray []any
		if err := json.Unmarshal(data, &rawArray); err != nil {
			return err
		}

		normalized, _ := json.Marshal(rawArray)
		return json.Unmarshal(normalized, v)
	}

	normalizeTimeFields(raw)

	normalized, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	return json.Unmarshal(normalized, v)
}

func normalizeTimeFields(data any) {
	switch v := data.(type) {
	case map[string]any:
		for key, value := range v {
			if str, ok := value.(string); ok {

				if normalized := tryParseTime(str); normalized != "" {
					v[key] = normalized
				}
			} else {
				normalizeTimeFields(value)
			}
		}
	case []any:
		for _, item := range v {
			normalizeTimeFields(item)
		}
	}
}

func tryParseTime(s string) string {
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {

			return t.Format(time.RFC3339)
		}
	}

	return ""
}
