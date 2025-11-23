# dockerfile
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git build-base

WORKDIR /app

# 依存関係だけ先にコピーしてキャッシュを効かせる
COPY go.mod go.sum ./
RUN go mod download

# ソースをコピーしてビルド
COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# デフォルトの main パッケージを ./cmd/cli に設定
ARG MAIN=./cmd/cli
RUN go build -ldflags="-s -w" -o /usr/local/bin/app ${MAIN}

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
RUN addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /usr/local/bin/app /usr/local/bin/app

USER app
ENV PORT=80
EXPOSE 80

ENTRYPOINT ["/usr/local/bin/app"]
