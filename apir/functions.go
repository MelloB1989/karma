package apir

import (
	"context"
	"log"
	"net/http"
)

func (client *APIClient) Get(endpoint string, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		if client.DebugMode {
			log.Printf("[APIR] Error in GET request: %v\n", err)
		}
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) GetWithContext(ctx context.Context, endpoint string, responseStruct any) error {
	respBody, err := client.sendRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		if client.DebugMode {
			log.Printf("[APIR] Error in GET request: %v\n", err)
		}
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) Post(endpoint string, requestBody, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodPost, endpoint, requestBody)
	if err != nil {
		if client.DebugMode {
			log.Printf("[APIR] Error in POST request: %v\n", err)
		}
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) PostWithContext(ctx context.Context, endpoint string, requestBody, responseStruct any) error {
	respBody, err := client.sendRequestWithContext(ctx, http.MethodPost, endpoint, requestBody)
	if err != nil {
		if client.DebugMode {
			log.Printf("[APIR] Error in POST request: %v\n", err)
		}
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) Put(endpoint string, requestBody, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodPut, endpoint, requestBody)
	if err != nil {
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) PutWithContext(ctx context.Context, endpoint string, requestBody, responseStruct any) error {
	respBody, err := client.sendRequestWithContext(ctx, http.MethodPut, endpoint, requestBody)
	if err != nil {
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) Delete(endpoint string, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) DeleteWithContext(ctx context.Context, endpoint string, responseStruct any) error {
	respBody, err := client.sendRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) Patch(endpoint string, requestBody, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodPatch, endpoint, requestBody)
	if err != nil {
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) PatchWithContext(ctx context.Context, endpoint string, requestBody, responseStruct any) error {
	respBody, err := client.sendRequestWithContext(ctx, http.MethodPatch, endpoint, requestBody)
	if err != nil {
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) Options(endpoint string, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodOptions, endpoint, nil)
	if err != nil {
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) Head(endpoint string, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodHead, endpoint, nil)
	if err != nil {
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) Connect(endpoint string, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodConnect, endpoint, nil)
	if err != nil {
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) Trace(endpoint string, responseStruct any) error {
	respBody, err := client.sendRequest(http.MethodTrace, endpoint, nil)
	if err != nil {
		return err
	}

	return client.unmarshalResponse(respBody, responseStruct)
}

func (client *APIClient) AddHeader(key, value string) {
	if client.Headers == nil {
		client.Headers = make(map[string]string)
	}
	client.Headers[key] = value
}

func (client *APIClient) RemoveHeader(key string) {
	delete(client.Headers, key)
}

func (client *APIClient) GetHeaders() map[string]string {
	if client.Headers == nil {
		return make(map[string]string)
	}
	headers := make(map[string]string, len(client.Headers))
	for k, v := range client.Headers {
		headers[k] = v
	}
	return headers
}

func IsHTTPError(err error, statusCode int) bool {
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr.StatusCode == statusCode
	}
	return false
}

func GetHTTPError(err error) (*HTTPError, bool) {
	httpErr, ok := err.(*HTTPError)
	return httpErr, ok
}
