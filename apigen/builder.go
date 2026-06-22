package apigen

// Fluent builder ------------------------------------------------------------
//
// The builder is the recommended, low-boilerplate way to declare an API.
// Struct examples are generated automatically (see examples.go), so most
// endpoints need no hand-written JSON at all:
//
//	api := apigen.New("Raftaar API", "Rider/driver platform").
//		Servers("https://api.example.com").
//		Var("api_version", "v1")
//
//	api.POST("/auth/login", "Login with phone").
//		Desc("Sends an OTP to the given phone number.").
//		Body(LoginBody{}).
//		OK(LoginSuccess{}, "OTP sent").
//		Fail(400, "Invalid phone number", ErrorResponse{}).
//		Add()
//
//	if err := api.ExportAll(); err != nil { /* handle */ }

// New starts a new API definition with sensible defaults: docs are written to
// ./docs with a file name derived from the API name. Override with Output().
func New(name, description string) *APIDefinition {
	return &APIDefinition{
		Name:            name,
		Description:     description,
		BaseURLs:        []string{},
		GlobalVariables: make(map[string]string),
		Endpoints:       []Endpoint{},
		OutputFolder:    "docs",
	}
}

// Servers appends base URLs the API is served from.
func (api *APIDefinition) Servers(urls ...string) *APIDefinition {
	api.BaseURLs = append(api.BaseURLs, urls...)
	return api
}

// Output sets the folder and base file name for exported docs.
func (api *APIDefinition) Output(folder, baseName string) *APIDefinition {
	api.OutputFolder = folder
	api.OutputFileBaseName = baseName
	return api
}

// Var adds a global variable (alias of AddGlobalVariable).
func (api *APIDefinition) Var(name, value string) *APIDefinition {
	return api.AddGlobalVariable(name, value)
}

// Err returns the first error encountered while building endpoints via the
// fluent API, if any. ExportAll also returns it, so checking there is enough.
func (api *APIDefinition) Err() error { return api.buildErr }

func (api *APIDefinition) note(err error) {
	if err != nil && api.buildErr == nil {
		api.buildErr = err
	}
}

// Route accumulates one endpoint. Create it with api.GET/POST/etc., chain
// configuration, and finish with Add().
type Route struct {
	api *APIDefinition
	ep  Endpoint
}

func (api *APIDefinition) route(method, path, summary string) *Route {
	return &Route{api: api, ep: Endpoint{Method: method, Path: path, Summary: summary}}
}

// Route starts building an endpoint for an arbitrary HTTP method. Useful when
// the method is only known at runtime; otherwise prefer GET/POST/etc.
func (api *APIDefinition) Route(method, path, summary string) *Route {
	return api.route(method, path, summary)
}

// GET starts building a GET endpoint.
func (api *APIDefinition) GET(path, summary string) *Route { return api.route("GET", path, summary) }

// POST starts building a POST endpoint.
func (api *APIDefinition) POST(path, summary string) *Route { return api.route("POST", path, summary) }

// PUT starts building a PUT endpoint.
func (api *APIDefinition) PUT(path, summary string) *Route { return api.route("PUT", path, summary) }

// PATCH starts building a PATCH endpoint.
func (api *APIDefinition) PATCH(path, summary string) *Route {
	return api.route("PATCH", path, summary)
}

// DELETE starts building a DELETE endpoint.
func (api *APIDefinition) DELETE(path, summary string) *Route {
	return api.route("DELETE", path, summary)
}

// Desc sets the long-form description.
func (r *Route) Desc(d string) *Route { r.ep.Description = d; return r }

// Auth marks the endpoint as requiring authentication of the given type
// (e.g. "bearer", "apiKey"). The description is optional.
func (r *Route) Auth(kind string, description ...string) *Route {
	a := &Auth{Type: kind}
	if len(description) > 0 {
		a.Description = description[0]
	}
	r.ep.Authentication = a
	return r
}

// Bearer is shorthand for Auth("bearer", ...).
func (r *Route) Bearer(description ...string) *Route { return r.Auth("bearer", description...) }

// Header documents a request header and its example value.
func (r *Route) Header(key Headers, example string) *Route {
	if r.ep.Headers == nil {
		r.ep.Headers = map[Headers]string{}
	}
	r.ep.Headers[key] = example
	return r
}

// Query documents a query parameter.
func (r *Route) Query(name, description string, opts ...ParamOption) *Route {
	p := Parameter{Name: name, Type: "string", Description: description}
	for _, o := range opts {
		o(&p)
	}
	r.ep.QueryParams = append(r.ep.QueryParams, p)
	return r
}

// PathParam describes a path parameter richly. Parameters present in the path
// are auto-detected on Add even without this; use it to add a type/example.
func (r *Route) PathParam(name, description string, opts ...ParamOption) *Route {
	p := Parameter{Name: name, Type: "string", Required: true, Description: description}
	for _, o := range opts {
		o(&p)
	}
	r.ep.PathParams = append(r.ep.PathParams, p)
	return r
}

// Body sets the JSON request body from a struct. Field examples are generated
// automatically; pass overrides to tweak specific fields.
func (r *Route) Body(v any, overrides ...FieldOverride) *Route {
	body, err := RequestBodyFromStruct(v, ContentTypeJSON, true, overrides)
	r.api.note(err)
	if err == nil {
		r.ep.RequestBody = body
	}
	return r
}

// Response documents a response whose JSON body is generated from a struct.
// Pass nil for body when the response has no content.
func (r *Route) Response(status int, description string, body any, opts ...RespOption) *Route {
	var resp Response
	if body == nil {
		resp = Response{StatusCode: status, Description: description}
	} else {
		built, err := ResponseFromStruct(status, description, body, ContentTypeJSON, nil)
		r.api.note(err)
		if err != nil {
			return r
		}
		resp = *built
	}
	for _, o := range opts {
		o(&resp)
	}
	r.ep.Responses = append(r.ep.Responses, resp)
	return r
}

// OK adds a 200 response. description defaults to "OK".
func (r *Route) OK(body any, description ...string) *Route {
	return r.Response(200, descOr(description, "OK"), body)
}

// Created adds a 201 response. description defaults to "Created".
func (r *Route) Created(body any, description ...string) *Route {
	return r.Response(201, descOr(description, "Created"), body)
}

// NoContent adds a 204 response with no body.
func (r *Route) NoContent(description ...string) *Route {
	return r.Response(204, descOr(description, "No Content"), nil)
}

// Fail adds an error response. Pass nil body for no content.
func (r *Route) Fail(status int, description string, body any, opts ...RespOption) *Route {
	return r.Response(status, description, body, opts...)
}

// Add finalizes the route, appends it to the API definition, and returns the
// API for further chaining.
func (r *Route) Add() *APIDefinition {
	r.api.AddEndpoint(r.ep)
	return r.api
}

// ParamOption customizes a Parameter built by Query/PathParam.
type ParamOption func(*Parameter)

// ParamRequired marks a parameter as required.
func ParamRequired() ParamOption { return func(p *Parameter) { p.Required = true } }

// ParamType sets a parameter's type (default "string").
func ParamType(t string) ParamOption { return func(p *Parameter) { p.Type = t } }

// ParamExample sets a parameter's example value.
func ParamExample(ex string) ParamOption { return func(p *Parameter) { p.Example = ex } }

// RespOption customizes a Response built by Response/OK/Fail/etc.
type RespOption func(*Response)

// RespHeader documents a response header and its example value.
func RespHeader(key Headers, example string) RespOption {
	return func(resp *Response) {
		if resp.Headers == nil {
			resp.Headers = map[Headers]string{}
		}
		resp.Headers[key] = example
	}
}

func descOr(provided []string, fallback string) string {
	if len(provided) > 0 && provided[0] != "" {
		return provided[0]
	}
	return fallback
}
