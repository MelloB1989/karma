package apir

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func (client *APIClient) Get(endpoint string, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		if client.DebugMode {
			log.Printf("Error in GET request: %v\n", err)
		}
		return err
	}

	return json.Unmarshal(respBody, responseStruct)
}

func (client *APIClient) Post(endpoint string, requestBody, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodPost, endpoint, requestBody)
	if err != nil {
		if client.DebugMode {
			log.Printf("Error in GET request: %v\n", err)
		}
		return err
	}

	return json.Unmarshal(respBody, responseStruct)
}

func (client *APIClient) Put(endpoint string, requestBody, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodPut, endpoint, requestBody)
	if err != nil {
		return err
	}

	return json.Unmarshal(respBody, responseStruct)
}

func (client *APIClient) Delete(endpoint string, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}

	return json.Unmarshal(respBody, responseStruct)
}

func (client *APIClient) Patch(endpoint string, requestBody, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodPatch, endpoint, requestBody)
	if err != nil {
		return err
	}

	return json.Unmarshal(respBody, responseStruct)
}

func (client *APIClient) Options(endpoint string, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodOptions, endpoint, nil)
	if err != nil {
		return err
	}

	return json.Unmarshal(respBody, responseStruct)
}

func (client *APIClient) Head(endpoint string, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodHead, endpoint, nil)
	if err != nil {
		return err
	}

	return json.Unmarshal(respBody, responseStruct)
}

func (client *APIClient) Connect(endpoint string, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodConnect, endpoint, nil)
	if err != nil {
		return err
	}

	return json.Unmarshal(respBody, responseStruct)
}

func (client *APIClient) Trace(endpoint string, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodTrace, endpoint, nil)
	if err != nil {
		return err
	}

	return json.Unmarshal(respBody, responseStruct)
}

func (client *APIClient) AddHeader(key, value string) {
	client.Headers[key] = value
}

func (client *APIClient) RemoveHeader(key string) {
	delete(client.Headers, key)
}

func (client *APIClient) UploadFile(endpoint, fieldName, filePath string, additionalFields map[string]string, responseStruct any) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create a buffer to store the multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a form file field
	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy the file content to the form field
	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add additional fields to the form if provided
	for key, value := range additionalFields {
		err = writer.WriteField(key, value)
		if err != nil {
			return fmt.Errorf("failed to add field %s: %w", key, err)
		}
	}

	// Close the writer before making the request
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create the request
	url := fmt.Sprintf("%s%s", client.BaseURL, endpoint)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range client.Headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Create HTTP client with a timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Send the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Unmarshal response if responseStruct is provided
	if responseStruct != nil {
		return json.Unmarshal(respBody, responseStruct)
	}

	return nil
}
