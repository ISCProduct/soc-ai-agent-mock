# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git build-base

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server application to /bin directory
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /bin/server ./cmd/server

# Build the large data seed command
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /bin/seed-large ./cmd/seed-large

# Runtime stage
FROM golang:1.25-alpine

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates git build-base tzdata

# Copy source code (for potential hot reload via volume mount)
COPY --from=builder /app /app

# Copy binary from builder to /bin (won't be overwritten by volume mount)
COPY --from=builder /bin/server /bin/server
COPY --from=builder /bin/seed-large /bin/seed-large

# Make binary executable
RUN chmod +x /bin/server /bin/seed-large

# Expose port
EXPOSE 8080

# Run the application from /bin
CMD ["/bin/server"]
