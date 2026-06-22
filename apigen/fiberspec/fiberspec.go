// Package fiberspec is the bridge between a Fiber server and apigen docs.
//
// Each endpoint declares a Spec next to its handler and registers it through a
// Registry. Mounting records the Spec for documentation AND wires the route, so
// the docs and the running server can never drift apart. A separate docs
// command can build the same Registry with a nil auth handler and call Build()
// to emit JSON/OpenAPI/Postman/Markdown.
//
//	reg := fiberspec.New(cognitoMiddleware) // auth handler applied to Auth:true specs
//
//	reg.Mount(app, fiberspec.Spec{
//		Method: "GET", Path: "/api/me", Summary: "Echo claims", Auth: true,
//		Responses: []fiberspec.Response{{StatusCode: 200, Description: "OK", Body: MeResponse{}}},
//	}, h.Me)
//
//	def, err := reg.Build("My API", "desc", []string{"https://api.example.com"}, "docs", "api")
//	_ = def.ExportAll()
//
// The Registry holds no global state, so multiple APIs can coexist in one
// process and tests can build throwaway registries freely.
package fiberspec

import (
	"github.com/MelloB1989/karma/apigen"
	"github.com/gofiber/fiber/v2"
)

// Spec documents and wires one endpoint. Path is always the full path a client
// calls (e.g. "/api/me"); auth is applied per-route, so there is no route-group
// prefix to reconcile.
type Spec struct {
	Method      string // "GET", "POST", "PUT", "PATCH", "DELETE", ...
	Path        string // full path, e.g. "/api/riders/profile"
	Summary     string
	Description string
	Auth        bool                      // true => the Registry's auth handler runs before this route
	Headers     map[apigen.Headers]string // request headers to document (name -> example), optional
	Request     any                       // zero value of the request body struct, or nil
	Responses   []Response
}

// Response documents one possible response for an endpoint.
type Response struct {
	StatusCode  int
	Description string
	Body        any                       // zero value of the response body struct, or nil
	Headers     map[apigen.Headers]string // response headers to document (name -> example), optional
}

// Registry collects mounted specs and knows the auth middleware to apply to
// authenticated routes. Create one with New.
type Registry struct {
	auth  fiber.Handler
	specs []Spec
}

// New returns a Registry. auth is the middleware applied (before the handler)
// to every Spec with Auth=true; pass nil when only building docs.
func New(auth fiber.Handler) *Registry {
	return &Registry{auth: auth}
}

// Mount records s for documentation and registers it on router. For an
// authenticated spec the Registry's auth handler is prepended to the handler
// chain. router is typically the *fiber.App, but any fiber.Router works.
func (r *Registry) Mount(router fiber.Router, s Spec, handlers ...fiber.Handler) {
	r.specs = append(r.specs, s)

	chain := handlers
	if s.Auth && r.auth != nil {
		chain = append([]fiber.Handler{r.auth}, handlers...)
	}
	router.Add(s.Method, s.Path, chain...)
}

// Specs returns every spec mounted so far.
func (r *Registry) Specs() []Spec {
	return r.specs
}

// Build converts the mounted specs into an apigen.APIDefinition via the fluent
// builder. Field examples are generated automatically from the struct types, so
// handlers never hand-write example JSON.
func (r *Registry) Build(name, description string, servers []string, outputFolder, outputFile string) (*apigen.APIDefinition, error) {
	def := apigen.New(name, description).
		Servers(servers...).
		Output(outputFolder, outputFile)

	for _, s := range r.specs {
		rt := def.Route(s.Method, s.Path, s.Summary).Desc(s.Description)

		if s.Auth {
			rt.Bearer()
		}
		for key, example := range s.Headers {
			rt.Header(key, example)
		}
		if s.Request != nil {
			rt.Body(s.Request)
		}

		for _, resp := range s.Responses {
			var opts []apigen.RespOption
			for key, example := range resp.Headers {
				opts = append(opts, apigen.RespHeader(key, example))
			}
			rt.Response(resp.StatusCode, resp.Description, resp.Body, opts...)
		}

		rt.Add()
	}

	return def, def.Err()
}
