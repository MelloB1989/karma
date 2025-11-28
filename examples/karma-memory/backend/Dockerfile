FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy the entire repository to preserve the directory structure
# required by the 'replace' directive in go.mod
COPY . .

WORKDIR /app/examples/karma-ui/backend

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

# Final stage
FROM alpine:latest

WORKDIR /root/

# Install ca-certificates for HTTPS requests (needed for OpenAI/Groq/etc)
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/examples/karma-ui/backend/server .

# Expose the port
EXPOSE 8080

# Run the server
CMD ["./server"]
