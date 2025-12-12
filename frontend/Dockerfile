# Build stage
FROM node:20-alpine AS builder
WORKDIR /app

# パッケージのインストール
COPY package*.json ./
RUN npm install

# ソースコード全体をコピー
COPY . .

# ビルド
RUN npm run build

# Production stage
FROM node:20-alpine
WORKDIR /app

ENV NODE_ENV=production

# 本番環境にもソースコードとビルド成果物を含める
COPY --from=builder /app /app

# .env が必要であれば下記を有効化（ECSなどでSecrets経由推奨）
# COPY .env .env

# Next.js の本番サーバを起動
EXPOSE 3000
CMD ["npm", "start"]
