# ----------------------------------------------------------------------------
# 1. ビルダー・ステージ
# Next.jsアプリケーションのビルド（.nextディレクトリ生成）を担当
# ----------------------------------------------------------------------------
FROM node:20-alpine AS builder

# 作業ディレクトリの設定
WORKDIR /app

# 依存関係ファイル (package.json, lockfile) のみ先にコピーしてキャッシュを活用
COPY package*.json ./
RUN npm install

# アプリケーションソースコード全体をコピー
COPY . .

# Next.jsのプロダクションビルドを実行
# next.config.jsで standalone: true が設定されていることを前提とします
RUN npm run build

# ----------------------------------------------------------------------------
# 2. プロダクション・ステージ
# 実行に必要な最小限のファイルのみをコピーし、軽量でセキュアな環境を構築
# ----------------------------------------------------------------------------
FROM node:20-alpine AS runner

# プロダクション環境を指定
ENV NODE_ENV production
WORKDIR /app

# セキュリティ向上のため、非rootユーザーを使用（Next.jsの推奨設定）
# UID 1001 の nodejs グループと nextjs ユーザーを作成
RUN addgroup -g 1001 -S nodejs && \
    adduser -S nextjs -u 1001 -G nodejs
USER nextjs

# Standaloneモードの成果物をコピー
# これにはサーバー実行に必要なコードと依存関係が含まれます
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./

# 静的アセットと public ディレクトリをコピー
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
COPY --from=builder --chown=nextjs:nodejs /app/public ./public

# アプリケーションがリッスンするポート
EXPOSE 3000

# Standaloneモードのサーバー起動
CMD ["node", "server.js"]