# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Go 1.25 以上を要求するモジュールに対応するための設定
ENV GOTOOLCHAIN=auto

# Install build dependencies
RUN apk add --no-cache git build-base

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build main application
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /bin/server ./cmd/server

# Build seed script (optional)
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /bin/seed-large ./cmd/seed-large

# Runtime stage
FROM golang:1.21-alpine

WORKDIR /app

# Install runtime dependencies (if needed by app)
RUN apk --no-cache add ca-certificates git tzdata

# Optional: copy full source for debugging/hot reload
COPY --from=builder /app /app

# Copy built binaries
COPY --from=builder /bin/server /bin/server
COPY --from=builder /bin/seed-large /bin/seed-large

# Make them executable
RUN chmod +x /bin/server /bin/seed-large

EXPOSE 8080

CMD ["/bin/server"]
