# SOC AI Agent Mock

小規模のフルスタックリポジトリ（フロントエンド + バックエンド）。
このリポジトリは、求人／企業マッチングや会話型 AI を組み合わせたプロトタイプ的な実装です。

**重要**: この README は開発メンバー向けのセットアップ手順・依存関係・テスト・品質管理をまとめたものです。

**開発メンバー**

- **バックエンド**: 大橋 和幸
- **フロントエンド**: 原 拓哉

**概要**

- **目的**: 求人／企業マッチングのプロトタイプと、OpenAI を用いた会話エージェントの統合を示す。フロントは Next.js、バックエンドは Go（GORM + MySQL）で構成。

**ディレクトリ構成（抜粋）**

- **`Backend/`**: Go サービス。`cmd/` に複数の実行用エントリ（`server`、`api`、`migrate`、`seed` 等）、`internal/` に設定・コントローラ・リポジトリ等。
- **`frontend/`**: Next.js (TypeScript) アプリケーション。`app/` にページ・API など。
- **`compose.yml`**: Docker / Compose 用の定義（ローカルでコンテナで立ち上げたいときに使用）。
- **`mysql/`**: MySQL 設定（ローカル Docker 用の conf などが含まれる可能性あり）。

**技術スタック**

- **Backend**: Go (module: `go 1.25`), GORM (MySQL driver), `github.com/sashabaranov/go-openai` を利用して OpenAI と連携。
- **Frontend**: Next.js 16 / React 19 / TypeScript / MUI / Radix UI / React Flow。
- **CI / 実行環境**: Docker / Docker Compose（`compose.yml`）、Playwright による E2E テスト（フロント）など。

**必須前提ソフトウェア**

- macOS / Linux / Windows + WSL
- Go >= 1.25
- Node.js (推奨: 18+、Next.js 16 の要件に合わせる)
- npm / pnpm / yarn（ここでは `npm` を例示）
- Docker / Docker Compose（オプション、コンテナで動かす場合）

**環境変数（バックエンド） — 例 `.env`**

```env
# MySQL
DB_USER=app_user
DB_PASS=app_pass
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=app_db

# サーバー
SERVER_PORT=8080

# OpenAI
OPENAI_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
#  使用モデル
OPENAI_MODEL=omuni
```

※ `Backend/internal/config/config.go` で読み込まれる環境変数名を使用しています（デフォルト値あり）。

**ローカル開発: データベース（簡易）**

- Docker を使って MySQL を立てる例:

```sh
docker run --name soc-mysql -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=app_db -e MYSQL_USER=app_user -e MYSQL_PASSWORD=app_pass \
  -p 3306:3306 -d mysql:8.0
```

または、リポジトリの `compose.yml` を使って一括起動:

```sh
docker compose -f compose.yml up --build
```

**バックエンド: ローカル実行**

- 依存取得

```sh
cd Backend
go mod download
```

- マイグレーション（もし `cmd/migrate` がある場合）

```sh
# .env をセットした上で
go run ./cmd/migrate
```

- サーバ起動（開発）

```sh
SERVER_PORT=8080 go run ./cmd/server
# または API 実行バイナリが `cmd/api` にある場合:
go run ./cmd/api
```

**開発用シードデータ**

- シードが用意されている場合:

```sh
go run ./cmd/seed
go run ./cmd/seed-large # 大量データ用（注意: 時間がかかる）
```

**フロントエンド: ローカル実行**

- 依存インストール

```sh
cd frontend
npm install
```

- 開発サーバ起動

```sh
npm run dev
# ブラウザで http://localhost:3000 を確認
```

**フロントの E2E / テスト**

- Playwright テストは `frontend/e2e` にあります。インストール後に実行:

```sh
cd frontend
npx playwright install # 初回のみ
npx playwright test
```

**依存関係管理**

- バックエンド: Go Modules (`go.mod`) — `go mod download` / `go mod tidy`。
- フロントエンド: `package.json` / `npm install`。`npm run build` で本番ビルド。

**品質管理（推奨ワークフロー）**

- **Go コード**: `gofmt` / `go vet` を CI で実行。可能なら `golangci-lint` を導入。
- **フロント**: `npm run lint`（ESLint）、`prettier` を導入してフォーマットを統一。
- **テスト**: フロントの Playwright E2E を CI で走らせる。将来的にはバックエンドのユニットテスト追加を推奨。
- **PR ルール**: すべての PR に対して Lint と E2E を通過させる（GitHub Actions を推奨）。

**Docker / コンテナ化**

- ルートの `compose.yml` にサービス定義がある場合、これでフルスタックを起動可能。
- 個別ビルド

```sh
docker build -t soc-backend:local Backend
docker build -t soc-frontend:local frontend
```

**よくあるトラブルと対処**

- DB 接続エラー: `.env` の `DB_HOST` / `DB_PORT` / 資格情報を確認。MySQL がコンテナで動いているか確認。
- OpenAI キーがない: `OPENAI_API_KEY` を設定しないとバックエンドは初期化時にエラーを返します（`Backend/internal/openai` に依存）。
- フロントのビルド失敗: Node のバージョンを確認（推奨は安定した LTS または Next.js の要件に合わせる）。

**次のステップ（提案）**

- GitHub Actions による CI (Lint + Unit + Playwright E2E) の追加。
- `Makefile` や `justfile` を追加して開発コマンドを統一。
- バックエンドのユニットテスト追加（`internal/` のサービス層のテストなど）。

---
