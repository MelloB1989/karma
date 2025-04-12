# 📦 `apir` – Simple Go HTTP API Client

The `apir` package provides a clean and reusable HTTP client for interacting with REST APIs. It supports standard HTTP methods, automatic JSON (un)marshalling, customizable headers, and file uploads.

---

## 🔧 Installation

```bash
go get github.com/MelloB1989/karma/apir
```

---

## 📘 Usage

### ✅ Initialize the Client

```go
client := apir.NewAPIClient("https://api.example.com", map[string]string{
	"Authorization": "Bearer your_token",
	"Content-Type":  "application/json",
})
```

---

## 🧩 HTTP Methods

Each method sends a request and unmarshals the response JSON into `responseStruct`.

### `GET`

```go
err := client.Get("/endpoint", &responseStruct)
```

### `POST`

```go
err := client.Post("/endpoint", requestBody, &responseStruct)
```

### `PUT`

```go
err := client.Put("/endpoint", requestBody, &responseStruct)
```

### `DELETE`

```go
err := client.Delete("/endpoint", &responseStruct)
```

### `PATCH`

```go
err := client.Patch("/endpoint", requestBody, &responseStruct)
```

### `OPTIONS`

```go
err := client.Options("/endpoint", &responseStruct)
```

### `HEAD`

```go
err := client.Head("/endpoint", &responseStruct)
```

### `CONNECT`

```go
err := client.Connect("/endpoint", &responseStruct)
```

### `TRACE`

```go
err := client.Trace("/endpoint", &responseStruct)
```

---

## 📤 File Uploads

Upload a file via `multipart/form-data`.

```go
err := client.UploadFile(
	"/upload",
	"fileField",                      // Form field name
	"./path/to/file.png",            // Local file path
	map[string]string{               // Optional additional fields
		"description": "Test upload",
	},
	&responseStruct,
)
```

---

## 🧰 Header Management

### Add a Header

```go
client.AddHeader("X-Custom-Header", "value")
```

### Remove a Header

```go
client.RemoveHeader("X-Custom-Header")
```

---

## 📥 Internal Method

### `sendRequest`

Used internally to make HTTP requests. Not required for general use, but available if needed:

```go
responseBody, err := client.sendRequest("GET", "/endpoint", nil)
```

---

## 📌 Notes

- Automatically serializes `requestBody` to JSON.
- Automatically deserializes response into provided struct.
- Returns descriptive error if status code is not in the 2xx range.

---

## 🛡️ Example

```go
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

client := apir.NewAPIClient("https://api.example.com", nil)

var user User
err := client.Get("/users/1", &user)
if err != nil {
	log.Fatal(err)
}

fmt.Printf("User: %+v\n", user)
```
