package apigen

import "encoding/json"

// APIDefinition represents a complete API with base URL and endpoints
type APIDefinition struct {
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	BaseURLs           []string          `json:"baseUrls"`
	GlobalVariables    map[string]string `json:"globalVariables,omitempty"`
	Endpoints          []Endpoint        `json:"endpoints"`
	OutputFileBaseName string            `json:"-"` // Not serialized but used for exports
	OutputFolder       string            `json:"-"` // Not serialized but used for exports
}

type Headers string
type KarmaHeaders map[Headers]string

const (
	HeaderPrivateToken    Headers = "Private-Token"
	HeaderContentType     Headers = "Content-Type"
	HeaderAccept          Headers = "Accept"
	HeaderUserAgent       Headers = "User-Agent"
	HeaderAuthorization   Headers = "Authorization"
	HeaderCacheControl    Headers = "Cache-Control"
	HeaderContentLength   Headers = "Content-Length"
	HeaderContentEncoding Headers = "Content-Encoding"
	HeaderContentLanguage Headers = "Content-Language"
	HeaderContentLocation Headers = "Content-Location"
)

type ContentType string

const (
	ContentTypeJSON              ContentType = "application/json"
	ContentTypeXML               ContentType = "application/xml"
	ContentTypeFormURLEncoded    ContentType = "application/x-www-form-urlencoded"
	ContentTypeMultipartFormData ContentType = "multipart/form-data"
	ContentTypeTextPlain         ContentType = "text/plain"
	ContentTypeTextHTML          ContentType = "text/html"
	ContentTypeOctetStream       ContentType = "application/octet-stream"
	ContentTypeJPEG              ContentType = "image/jpeg"
	ContentTypePNG               ContentType = "image/png"
	ContentTypeGIF               ContentType = "image/gif"
)

// Endpoint represents a single API endpoint
type Endpoint struct {
	Path           string             `json:"path"`
	Method         string             `json:"method"`
	Summary        string             `json:"summary"`
	Description    string             `json:"description"`
	Headers        map[Headers]string `json:"headers,omitempty"`
	QueryParams    []Parameter        `json:"queryParams,omitempty"`
	PathParams     []Parameter        `json:"pathParams,omitempty"` // New field for path parameters
	RequestBody    *RequestBody       `json:"requestBody,omitempty"`
	Responses      []Response         `json:"responses"`
	Authentication *Auth              `json:"authentication,omitempty"`
}

// Parameter defines a request parameter (query or path param)
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
	Example     string `json:"example,omitempty"`
}

// RequestBody defines the structure of a request body
type RequestBody struct {
	ContentType ContentType        `json:"contentType"`
	Required    bool               `json:"required"`
	Schema      json.RawMessage    `json:"schema,omitempty"`
	Example     json.RawMessage    `json:"example,omitempty"`
	Fields      []RequestBodyField `json:"fields,omitempty"` // Structured field definitions
}

// RequestBodyField represents a single field in a request body
type RequestBodyField struct {
	Name        string             `json:"name"`
	JsonName    string             `json:"jsonName"`
	Type        string             `json:"type"`
	Required    bool               `json:"required"`
	Description string             `json:"description,omitempty"`
	Example     any                `json:"example,omitempty"`
	Fields      []RequestBodyField `json:"fields,omitempty"` // For nested objects
}

// Response defines a possible API response
type Response struct {
	StatusCode  int                `json:"statusCode"`
	Description string             `json:"description"`
	Headers     map[Headers]string `json:"headers,omitempty"`
	ContentType ContentType        `json:"contentType,omitempty"`
	Schema      json.RawMessage    `json:"schema,omitempty"`
	Example     json.RawMessage    `json:"example,omitempty"`
	Fields      []RequestBodyField `json:"fields,omitempty"` // Reusing the same field structure
}

// Auth defines authentication details
type Auth struct {
	Type        string            `json:"type"` // bearer, basic, apiKey, oauth2
	Description string            `json:"description,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

// FieldOverride allows overriding specific fields in a struct
type FieldOverride struct {
	Name        string `json:"name"`
	JsonName    string `json:"jsonName,omitempty"`
	Type        string `json:"type,omitempty"`
	Required    *bool  `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
	Example     any    `json:"example,omitempty"`
	Exclude     bool   `json:"exclude,omitempty"`
}
