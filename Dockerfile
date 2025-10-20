# Build stage
FROM golang:1.24-alpine AS builder

# Install minimal build dependencies
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application (pure Go)
RUN GOOS=linux go build -a -ldflags="-s -w" -o textanalyzer ./cmd/server

# Runtime stage
FROM alpine:3.20

# Install minimal runtime dependencies
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/textanalyzer .

# Create directory for database
RUN mkdir -p /data

# Expose port
EXPOSE 8080

# Set environment variables
ENV PORT=8080
ENV DB_PATH=/data/textanalyzer.db

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./textanalyzer", "-port", "8080", "-db", "/data/textanalyzer.db"]
