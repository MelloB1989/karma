package apir

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func (client *APIClient) UploadFile(endpoint, fieldName, filePath string, additionalFields map[string]string, responseStruct any) error {
	return client.UploadFileWithContext(context.Background(), endpoint, fieldName, filePath, additionalFields, responseStruct)
}

func (client *APIClient) UploadFileWithContext(ctx context.Context, endpoint, fieldName, filePath string, additionalFields map[string]string, responseStruct any) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err = io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	for key, value := range additionalFields {
		if err = writer.WriteField(key, value); err != nil {
			return fmt.Errorf("failed to add field %s: %w", key, err)
		}
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := fmt.Sprintf("%s%s", client.BaseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range client.Headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(respBody),
		}
	}

	if responseStruct != nil {
		return client.unmarshalResponse(respBody, responseStruct)
	}

	return nil
}

func (client *APIClient) UploadFiles(endpoint string, files map[string]string, additionalFields map[string]string, responseStruct any) error {
	return client.UploadFilesWithContext(context.Background(), endpoint, files, additionalFields, responseStruct)
}

func (client *APIClient) UploadFilesWithContext(ctx context.Context, endpoint string, files map[string]string, additionalFields map[string]string, responseStruct any) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for fieldName, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", filePath, err)
		}

		part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
		if err != nil {
			file.Close()
			return fmt.Errorf("failed to create form file for %s: %w", fieldName, err)
		}

		if _, err = io.Copy(part, file); err != nil {
			file.Close()
			return fmt.Errorf("failed to copy file content for %s: %w", fieldName, err)
		}
		file.Close()
	}

	for key, value := range additionalFields {
		if err := writer.WriteField(key, value); err != nil {
			return fmt.Errorf("failed to add field %s: %w", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := fmt.Sprintf("%s%s", client.BaseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range client.Headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(respBody),
		}
	}

	if responseStruct != nil {
		return client.unmarshalResponse(respBody, responseStruct)
	}

	return nil
}

func (client *APIClient) UploadReader(endpoint, fieldName, fileName string, reader io.Reader, additionalFields map[string]string, responseStruct any) error {
	return client.UploadReaderWithContext(context.Background(), endpoint, fieldName, fileName, reader, additionalFields, responseStruct)
}

func (client *APIClient) UploadReaderWithContext(ctx context.Context, endpoint, fieldName, fileName string, reader io.Reader, additionalFields map[string]string, responseStruct any) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err = io.Copy(part, reader); err != nil {
		return fmt.Errorf("failed to copy reader content: %w", err)
	}

	for key, value := range additionalFields {
		if err = writer.WriteField(key, value); err != nil {
			return fmt.Errorf("failed to add field %s: %w", key, err)
		}
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := fmt.Sprintf("%s%s", client.BaseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range client.Headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(respBody),
		}
	}

	if responseStruct != nil {
		return client.unmarshalResponse(respBody, responseStruct)
	}

	return nil
}
