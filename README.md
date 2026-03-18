# SOC AI Agent Mock

小規模のフルスタックリポジトリ（フロントエンド + バックエンド）。
このリポジトリは、求人／企業マッチングや会話型 AI を組み合わせたプロトタイプ的な実装です。

**重要**: この README は開発メンバー向けのセットアップ手順・依存関係・テスト・品質管理をまとめたものです。

**開発メンバー**

- **バックエンド**: 大橋 和幸
- **フロントエンド**: 原 拓哉

**概要**

- **目的**: 求人／企業マッチングのプロトタイプと、OpenAI を用いた会話エージェントの統合を示す。フロントは Next.js、バックエンドは Go（GORM + MySQL）で構成。

**ディレクトリ構成**

```
/
├── Backend/                 # Go バックエンドサービス
│   ├── cmd/                 # 実行エントリポイント（server / api / migrate / seed / crawl / cli）
│   ├── internal/            # 設定・コントローラ・サービス・リポジトリ等
│   └── storage/             # ローカルファイルストレージ（開発用）
├── frontend/                # Next.js (TypeScript) アプリケーション
│   ├── app/                 # ページ・API ルートハンドラ
│   ├── components/          # 共通コンポーネント
│   └── e2e/                 # Playwright E2E テスト
├── rag/                     # 職務経歴書レビュー RAG サービス（Python / FastAPI / CrewAI）
├── tools/
│   └── company-graph/       # 企業関係スクレイピングパイプライン（Go）
├── infra/                   # インフラ設定（ECS）
├── mcp/                     # MCP サーバー設定
├── mysql/                   # MySQL 設定（ローカル Docker 用）
└── compose.yml              # Docker Compose 定義
```

**技術スタック**

- **Backend**: Go 1.25 / GORM (MySQL driver) / AWS SDK v2 / `go-openai`
- **Frontend**: Next.js 16 / React 19 / TypeScript / MUI v7 / Radix UI / React Flow / Three.js
- **RAG**: Python / FastAPI / CrewAI / OpenAI Embeddings
- **CI / 実行環境**: Docker / Docker Compose / Playwright E2E

**主な機能**

| 機能 | 概要 |
|------|------|
| AI チャット分析 | OpenAI を用いた職種・興味分析と企業マッチング |
| 音声面接練習 | OpenAI Realtime API による会話 + 3D アバター（Three.js + wawa-lipsync） |
| 面接動画管理 | 面接動画を AWS S3 にアップロード・管理。管理者が Presigned URL で閲覧 |
| 職務経歴書レビュー | RAG サービス（DuckDuckGo + OpenAI Embeddings）によるフィードバック生成 |
| 企業関係図 | Mynavi / Rikunabi / CareerTasu / gBizINFO スクレイピング + React Flow 可視化 |
| 管理者ダッシュボード | ユーザー・企業・求人・クロール・監査ログの CRUD 管理 |
| OAuth 認証 | Google / GitHub OAuth2 + メール・パスワード認証 |
| メールレポート | 分析・面接レポートをメール配信 |

---

**必須前提ソフトウェア**

- macOS / Linux / Windows + WSL
- Go >= 1.25
- Node.js 18 以上（Next.js 16 の要件）
- npm / pnpm / yarn（ここでは `npm` を例示）
- Docker / Docker Compose（コンテナで動かす場合）

---

**環境変数（バックエンド） — 例 `.env`**

```env
# MySQL
DB_USER=app_user
DB_PASSWORD=app_pass
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=app_db

# サーバー
SERVER_PORT=8080

# OpenAI
OPENAI_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
OPENAI_MODEL=gpt-4o-mini

# Realtime（音声面接）
OPENAI_REALTIME_MODEL=gpt-realtime
OPENAI_REALTIME_VOICE=alloy
OPENAI_REALTIME_TRANSCRIBE_MODEL=gpt-4o-mini-transcribe
OPENAI_REALTIME_MAX_OUTPUT_TOKENS=120

# 面接レポート
INTERVIEW_REPORT_MODEL=gpt-4o-mini
INTERVIEW_TEMPLATE_VERSION=v1
INTERVIEW_MAX_MINUTES=10
INTERVIEW_MAX_COST_USD=1.8
INTERVIEW_COST_PER_MIN_USD=0.18

# gBizINFO
GBIZINFO_BASE_URL=https://api.biz-info.go.jp
GBIZINFO_API_KEY=xxxxxxxxxxxxxxxx

# OAuth2
BASE_URL=http://localhost:8080
GOOGLE_CLIENT_ID=xxxxxxxxxxxxxxxx
GOOGLE_CLIENT_SECRET=xxxxxxxxxxxxxxxx
GITHUB_CLIENT_ID=xxxxxxxxxxxxxxxx
GITHUB_CLIENT_SECRET=xxxxxxxxxxxxxxxx

# AWS S3（面接動画・職務経歴書）
AWS_REGION=ap-northeast-1
AWS_S3_BUCKET=your-bucket
AWS_S3_PREFIX=interview-videos
# AWS_ACCESS_KEY_ID=xxxxxxxxxxxxxxxxxxxx
# AWS_SECRET_ACCESS_KEY=yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy

# PDF アノテーション
# ANNOTATION_FONT_PATH=/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc

# RAG レビューサービス
RAG_REVIEW_URL=http://rag-review:9000
```

※ `Backend/internal/config/config.go` および各サービスで読み込まれる環境変数名を使用しています。

**環境変数（フロントエンド） — 例 `.env.local`**

```env
NEXT_PUBLIC_BACKEND_URL=http://localhost:80
NEXT_PUBLIC_INTERVIEW_MAX_MINUTES=10
NEXT_PUBLIC_INTERVIEW_MAX_COST_USD=1.8
NEXT_PUBLIC_INTERVIEW_COST_PER_MIN_USD=0.18
```

---

**ローカル開発: データベース（簡易）**

Docker を使って MySQL を単体で起動する例:

```sh
docker run --name soc-mysql -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=app_db -e MYSQL_USER=app_user -e MYSQL_PASSWORD=app_pass \
  -p 3306:3306 -d mysql:8.0
```

または、リポジトリの `compose.yml` を使って一括起動:

```sh
docker compose up -d --build
```

---

**バックエンド: ローカル実行**

```sh
cd Backend

# 依存取得
go mod download

# マイグレーション
go run ./cmd/migrate

# サーバ起動（開発）
go run ./cmd/server

# クローリング単発実行
go run ./cmd/crawl
```

**開発用シードデータ**

```sh
go run ./cmd/seed
go run ./cmd/seed-large  # 大量データ用（時間がかかる）
```

---

**フロントエンド: ローカル実行**

```sh
cd frontend

# 依存インストール
npm install

# 開発サーバ起動
npm run dev
# ブラウザで http://localhost:3000 を確認

# 本番ビルド
npm run build
```

---

**フロントの E2E / テスト**

Playwright テストは `frontend/e2e` にあります。

```sh
cd frontend
npx playwright install  # 初回のみ
npx playwright test
```

---

**Docker Compose サービス構成**

| サービス | 役割 | ポート | プロファイル |
|---------|------|--------|------------|
| `app` | Go バックエンド API | 8080 | （常時） |
| `db` | MySQL 8.0 | 3306 | （常時） |
| `frontend` | Next.js フロントエンド | 3000 | （常時） |
| `company-graph` | 企業スクレイピングパイプライン | 9100 | `company-graph` |
| `rag-review` | 職務経歴書 RAG レビュー | 9000 | `rag` |

**重いサービスを後回しにする手順**

`rag-review` は依存が重いため、初回は後回しにできます。

```sh
# まず軽量サービスのみ起動
docker compose up -d --build

# 後から rag-review をビルド・起動
docker compose --profile rag build rag-review
docker compose --profile rag up -d rag-review

# 企業グラフサービスを起動する場合
docker compose --profile company-graph up -d company-graph
```

**rag-review の依存バージョン固定（`rag/constraints.txt`）**

`pip install` が遅い / 失敗する場合はこちらを使用してください:

```
embedchain==0.1.114
langchain==0.2.17
langchain-community==0.2.19
langchain-core==0.2.43
langchain-openai==0.1.25
langchain-text-splitters==0.2.4
chromadb==0.4.24
litellm==1.44.22
openai==1.40.6
tiktoken==0.7.0
```

---

**主要 API エンドポイント一覧**

| カテゴリ | エンドポイント例 | 概要 |
|---------|----------------|------|
| 認証 | `POST /api/auth/register` | メール登録 |
| 認証 | `POST /api/auth/login` | ログイン |
| 認証 | `GET /api/auth/google` | Google OAuth 開始 |
| 認証 | `GET /api/auth/github` | GitHub OAuth 開始 |
| チャット | `POST /api/chat/messages` | チャットメッセージ送信 |
| チャット | `GET /api/chat/scores` | 分析スコア取得 |
| チャット | `POST /api/chat/send-report` | メールレポート送信 |
| 面接 | `POST /api/interviews` | セッション作成 |
| 面接 | `POST /api/interviews/{id}/start` | セッション開始 |
| 面接 | `POST /api/interviews/{id}/upload-video` | 動画 S3 アップロード（32MB 上限） |
| 面接 | `POST /api/interviews/{id}/send-report` | レポートメール送信 |
| Realtime | `GET /api/realtime/token` | WebRTC トークン取得 |
| 職務経歴書 | `POST /api/resume/upload` | 職務経歴書アップロード |
| 職務経歴書 | `GET /api/resume/review` | RAG レビュー取得 |
| 企業 | `GET /api/companies` | 企業一覧・検索 |
| 管理者 | `GET /api/admin/interviews` | 面接セッション一覧 |
| 管理者 | `GET /api/admin/interviews/{id}/videos/{vid}/url` | 動画 Presigned URL（15分有効） |
| 管理者 | `GET /api/admin/audit-logs` | 監査ログ閲覧 |
| 管理者 | `POST /api/admin/crawl/run` | クロール実行 |

---

**面接練習デモ手順（10分）**

1. バックエンドとフロントを起動
2. ログイン後、左メニューの「面接練習」を開く
3. `Start` を押して面接を開始（音声が出るのでミュートに注意）
4. 10分またはコスト上限で自動終了。`Stop` でも終了可能
5. 終了後、レポートが生成されるまで数十秒待つ

---

**品質管理（推奨ワークフロー）**

- **Go コード**: `gofmt` / `go vet` を CI で実行。可能なら `golangci-lint` を導入。
- **フロント**: `npm run lint`（ESLint）/ `prettier` でフォーマット統一。
- **テスト**: フロントの Playwright E2E を CI で走らせる。バックエンドのユニットテスト追加を推奨。
- **PR ルール**: すべての PR に対して Lint と E2E を通過させる（GitHub Actions 推奨）。

---

**よくあるトラブルと対処**

- **DB 接続エラー**: `.env` の `DB_HOST` / `DB_PORT` / 資格情報を確認。MySQL がコンテナで動いているか確認。
- **OpenAI キーがない**: `OPENAI_API_KEY` を設定しないとバックエンドは初期化時にエラーを返します。
- **OAuth が動かない**: `BASE_URL` / `GOOGLE_CLIENT_ID` / `GITHUB_CLIENT_ID` 等を確認。コールバック URL がプロバイダー側の設定と一致しているか確認。
- **S3 アップロード失敗**: `AWS_REGION` / `AWS_S3_BUCKET` と IAM 権限（`s3:PutObject` / `s3:GetObject`）を確認。
- **フロントのビルド失敗**: Node バージョンを確認（18 以上推奨）。
- **rag-review が起動しない**: `pip install` 失敗の場合は `rag/constraints.txt` の固定バージョンを使用。

---
