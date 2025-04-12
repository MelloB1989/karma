package apir

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type APIClient struct {
	BaseURL   string
	Headers   map[string]string
	DebugMode bool
}

func NewAPIClient(baseURL string, headers map[string]string) *APIClient {
	return &APIClient{
		BaseURL:   baseURL,
		Headers:   headers,
		DebugMode: false,
	}
}

func (client *APIClient) SetDebugMode(debug bool) {
	client.DebugMode = debug
}

func (client *APIClient) sendRequest(method, endpoint string, body any) ([]byte, error) {
	url := fmt.Sprintf("%s%s", client.BaseURL, endpoint)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	for key, value := range client.Headers {
		req.Header.Set(key, value)
	}

	if client.DebugMode {
		log.Printf("Request Method: %s", method)
		log.Printf("Request URL: %s", url)
		if body != nil {
			log.Printf("Request Body: %s", reqBody)
		}
		for key, value := range client.Headers {
			log.Printf("Request Header: %s: %s", key, value)
		}
	}

	clientHTTP := &http.Client{}
	resp, err := clientHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if client.DebugMode {
		log.Printf("Response Status: %s", resp.Status)
		log.Printf("Response Body: %s", respBody)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
