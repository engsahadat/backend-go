# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the application (modernc.org/sqlite works with CGO_ENABLED=0)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api ./cmd/api/main.go

# Final stage
FROM alpine:latest
WORKDIR /app

# Install root certificates for HTTPS requests (e.g. Gemini API)
RUN apk --no-cache add ca-certificates tzdata

# Copy the compiled binary from the builder stage
COPY --from=builder /api ./api

# Create necessary directories
RUN mkdir -p uploads

# Expose the port the app runs on
EXPOSE 8080

# Command to run the application
CMD ["./api"]
