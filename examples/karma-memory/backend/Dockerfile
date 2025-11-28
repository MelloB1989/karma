# Use lightweight Alpine Linux as base image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    && rm -rf /var/cache/apk/*

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the pre-built binary from host
COPY backend .

# Copy configuration files
COPY .env .

# Make the binary executable
RUN chmod +x backend

# Create a non-root user for security
RUN addgroup -g 1001 appgroup && \
    adduser -D -u 1001 -G appgroup appuser && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port for Go app
EXPOSE 8080

# Run the Go app
CMD ["./backend"]
