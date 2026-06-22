package apigen

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type addr struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

type sampleUser struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	FullName  string    `json:"full_name"`
	Age       int       `json:"age"`
	Active    bool      `json:"active"`
	Score     float64   `json:"score" example:"4.5"`
	Tags      []string  `json:"tags"`
	Address   addr      `json:"address"`
	Website   string    `json:"website" fake:"{url}"`
	CreatedAt time.Time `json:"created_at"`
}

// TestExampleRealistic checks that gofakeit produces realistic, type-correct
// example values driven by field names and tags.
func TestExampleRealistic(t *testing.T) {
	resp, err := ResponseFromStruct(200, "ok", sampleUser{}, ContentTypeJSON, nil)
	if err != nil {
		t.Fatalf("ResponseFromStruct: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(resp.Example, &m); err != nil {
		t.Fatalf("example is not valid JSON: %v\n%s", err, resp.Example)
	}

	if email, _ := m["email"].(string); !strings.Contains(email, "@") {
		t.Errorf("email field not realistic: %q", m["email"])
	}
	if name, _ := m["full_name"].(string); name == "" || !strings.Contains(name, " ") {
		t.Errorf("full_name not realistic: %q", m["full_name"])
	}
	if _, ok := m["age"].(float64); !ok { // JSON numbers decode to float64
		t.Errorf("age should be a number, got %T", m["age"])
	}
	// example:"4.5" tag must be coerced to a number, not the string "4.5".
	if score, ok := m["score"].(float64); !ok || score != 4.5 {
		t.Errorf("score example tag not honored: %v (%T)", m["score"], m["score"])
	}
	if web, _ := m["website"].(string); !strings.HasPrefix(web, "http") {
		t.Errorf("website fake tag not honored: %q", m["website"])
	}
	if nested, ok := m["address"].(map[string]any); !ok {
		t.Errorf("address should be a nested object, got %T", m["address"])
	} else if city, _ := nested["city"].(string); city == "" {
		t.Errorf("nested address.city not populated")
	}
	if tags, ok := m["tags"].([]any); !ok || len(tags) == 0 {
		t.Errorf("tags should be a non-empty array, got %v", m["tags"])
	}
}

// TestExampleDeterministic ensures regenerating docs yields byte-identical
// examples, so committed docs don't churn on every build.
func TestExampleDeterministic(t *testing.T) {
	a, err := ResponseFromStruct(200, "ok", sampleUser{}, ContentTypeJSON, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ResponseFromStruct(200, "ok", sampleUser{}, ContentTypeJSON, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(a.Example) != string(b.Example) {
		t.Errorf("examples not deterministic:\n--- a ---\n%s\n--- b ---\n%s", a.Example, b.Example)
	}
}

// TestBuilderWithHeaders exercises the fluent builder and response headers.
func TestBuilderWithHeaders(t *testing.T) {
	api := New("Test API", "desc").Servers("https://api.test")

	api.POST("/users", "Create user").
		Desc("Creates a user.").
		Bearer("Cognito access token").
		Body(sampleUser{}).
		Created(sampleUser{}, "User created").
		Fail(400, "Validation failed", nil).
		Add()

	// Attach a response header via the option form (Response takes RespOptions;
	// the OK/Created/Fail shorthands take only a description).
	api.GET("/users/{id}", "Get user").
		Response(200, "Found", sampleUser{}, RespHeader(HeaderCacheControl, "max-age=60")).
		Fail(404, "Not found", nil).
		Add()

	if err := api.Err(); err != nil {
		t.Fatalf("builder error: %v", err)
	}
	if len(api.Endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(api.Endpoints))
	}

	// Path params should be auto-detected from "/users/{id}".
	get := api.Endpoints[1]
	if len(get.PathParams) != 1 || get.PathParams[0].Name != "id" {
		t.Errorf("path param not auto-detected: %+v", get.PathParams)
	}

	md := generateMarkdown(api)
	if !strings.Contains(md, "Headers:") || !strings.Contains(md, "Cache-Control") {
		t.Errorf("response headers not rendered in markdown:\n%s", md)
	}
	if !strings.Contains(md, "max-age=60") {
		t.Errorf("response header example value missing from markdown")
	}
	if !strings.Contains(md, "**Auth:** `bearer`") {
		t.Errorf("authentication not rendered in markdown:\n%s", md)
	}
}
