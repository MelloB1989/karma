# 📦 `apir` — Lightweight Go HTTP Client

A reusable HTTP client wrapper in Go for interacting with REST APIs. It supports:

- Standard HTTP methods (`GET`, `POST`, `PUT`, `DELETE`, etc.)
- JSON encoding/decoding of request/response bodies
- Customizable headers (`AddHeader`, `RemoveHeader`)
- File upload via `multipart/form-data` (`UploadFile`)
- Unified internal request handler (`sendRequest`)

Used like:
```go
client := apir.NewAPIClient("https://api.com", map[string]string{
    "Authorization": "Bearer token",
})

var res SomeResponse
err := client.Post("/endpoint", reqBody, &res)
```
