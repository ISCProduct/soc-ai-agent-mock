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

# Build the server application
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
RUN go build -ldflags="-s -w" -o /app/bin/server ./cmd/server

# Runtime stage
FROM golang:1.25-alpine

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates git build-base tzdata

# Copy binary and source (for migrations)
COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app /app

# Expose port
EXPOSE 80

# Run the application
CMD ["/app/server"]

