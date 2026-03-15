# Build stage
FROM golang:1.24-bullseye AS builder

WORKDIR /app

ENV GOTOOLCHAIN=auto

RUN apt-get update && apt-get install -y --no-install-recommends \
      git \
 && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/company-graph .

# Runtime stage
FROM debian:bullseye-slim

RUN sed -i 's|http://deb.debian.org|https://deb.debian.org|g' /etc/apt/sources.list \
 && sed -i 's|http://security.debian.org|https://security.debian.org|g' /etc/apt/sources.list \
 && apt-get update -o Acquire::Retries=3 \
 && apt-get install -y --no-install-recommends \
      ca-certificates \
      tzdata \
      chromium \
      chromium-driver \
 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /bin/company-graph /bin/company-graph

EXPOSE 9100

CMD ["/bin/company-graph"]
