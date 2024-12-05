package payments

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/MelloB1989/karma/config"
	"github.com/golang-jwt/jwt"
)

func Decode(kp string) (map[string]interface{}, error) {
	// Remove the first 3 characters
	if len(kp) <= 3 {
		return nil, fmt.Errorf("input string is too short")
	}
	base := kp[3:]

	// Add padding if necessary
	if len(base)%4 != 0 {
		base += strings.Repeat("=", 4-len(base)%4)
	}

	// Decode the base64 string
	decodedBytes, err := base64.StdEncoding.DecodeString(base)
	if err != nil {
		return nil, fmt.Errorf("base64 decoding failed: %w", err)
	}
	tokenString := string(decodedBytes)

	// Decode the JWT token
	// token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	// if err != nil {
	// 	return nil, fmt.Errorf("jwt parsing failed: %w", err)
	// }

	// Parse the JWT token
	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Make sure the token's algorithm is what you expect:
		return []byte(config.DefaultConfig().JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}

	return nil, fmt.Errorf("could not parse token claims")
}

func DecodeAPI(API string) (map[string]interface{}, error) {
	// Remove the first 3 characters
	if len(API) <= 3 {
		return nil, fmt.Errorf("input string is too short")
	}
	base := API[3:]

	// Add padding if necessary
	if len(base)%4 != 0 {
		base += strings.Repeat("=", 4-len(base)%4)
	}

	// Decode the base64 string
	decodedBytes, err := base64.StdEncoding.DecodeString(base)
	if err != nil {
		return nil, fmt.Errorf("base64 decoding failed: %w", err)
	}

	// Unmarshal the decoded bytes into a map
	var result map[string]interface{}
	err = json.Unmarshal(decodedBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return result, nil
}

func TriggerWebhook(url string) error {

	// Create a new POST request with the JSON payload
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set the content type to application/json
	req.Header.Set("Content-Type", "application/json")

	// Send the request using the default HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook triggered but received non-OK response: %s", resp.Status)
	}

	return nil
}
