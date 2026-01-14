# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

ENV GOTOOLCHAIN=local

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
FROM golang:1.24-alpine

WORKDIR /app

ENV GOTOOLCHAIN=local

# Install runtime dependencies (keep minimal to reduce image size and IO)
RUN apk --no-cache add ca-certificates tzdata

#
# NOTE: Hot-reload (air) is omitted to avoid Go toolchain version constraints.
#

# Optional: copy full source for debugging/hot reload
COPY --from=builder /app /app

# Copy built binaries
COPY --from=builder /bin/server /bin/server
COPY --from=builder /bin/migrate /bin/migrate
COPY --from=builder /bin/seed-large /bin/seed-large

# Make them executable
RUN chmod +x /bin/server /bin/migrate /bin/seed-large

EXPOSE 8080

CMD ["/bin/server"]
