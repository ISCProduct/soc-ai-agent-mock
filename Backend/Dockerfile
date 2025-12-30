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

# Build migrate script
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /bin/migrate ./cmd/migrate

# Build seed script (optional)
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /bin/seed-large ./cmd/seed-large

# Runtime stage
FROM golang:1.21-alpine

WORKDIR /app

# Go 1.25 以上を要求するモジュールに対応するための設定
ENV GOTOOLCHAIN=auto

# Install runtime dependencies (if needed by app)
RUN apk --no-cache add ca-certificates git tzdata build-base

# Hot reload tool (dev)
RUN go install github.com/air-verse/air@latest

# Keep config outside /app to avoid volume masking
RUN printf '%s\n' \
  'root = "/app"' \
  '' \
  '[build]' \
  '  cmd = "go build -o /tmp/server ./cmd/server"' \
  '  bin = "/tmp/server"' \
  '  include_ext = ["go", "tpl", "tmpl", "html"]' \
  '  exclude_dir = ["vendor", "tmp", "bin", "node_modules"]' \
  '  delay = 1000' \
  '' \
  '[log]' \
  '  time = true' \
  > /tmp/.air.toml

# Optional: copy full source for debugging/hot reload
COPY --from=builder /app /app

# Copy built binaries
COPY --from=builder /bin/server /bin/server
COPY --from=builder /bin/migrate /bin/migrate
COPY --from=builder /bin/seed-large /bin/seed-large

# Make them executable
RUN chmod +x /bin/server /bin/migrate /bin/seed-large

EXPOSE 8080

CMD ["sh", "-c", "if [ \"$NODE_ENV\" = \"development\" ]; then air -c /tmp/.air.toml; else /bin/server; fi"]
