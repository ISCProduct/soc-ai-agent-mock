# Terraform — ECS Fargate インフラ

soc-ai-agent のバックエンド/フロントエンドを AWS ECS Fargate で運用するための Terraform コードです。

## 構成

```
infra/terraform/
├── main.tf          # provider / backend / locals
├── variables.tf     # 変数定義
├── outputs.tf       # 出力値
├── vpc.tf           # VPC / サブネット / セキュリティグループ
├── ecr.tf           # ECR リポジトリ
├── ecs.tf           # ECS クラスター / タスク定義 / サービス
├── alb.tf           # ALB / ターゲットグループ / リスナー
├── rds.tf           # RDS MySQL（オプション）
├── iam.tf           # IAM ロール / ポリシー
├── cloudwatch.tf    # CloudWatch ロググループ
└── terraform.tfvars.example
```

## 前提条件

- Terraform >= 1.5.0
- AWS CLI 設定済み（または環境変数 `AWS_*` 設定）
- S3 バケット `soc-ai-tfstate` と DynamoDB テーブル `soc-ai-tfstate-lock` が作成済み

### S3 バケット / DynamoDB の初期作成

```bash
aws s3api create-bucket \
  --bucket soc-ai-tfstate \
  --region ap-northeast-1 \
  --create-bucket-configuration LocationConstraint=ap-northeast-1

aws s3api put-bucket-versioning \
  --bucket soc-ai-tfstate \
  --versioning-configuration Status=Enabled

aws dynamodb create-table \
  --table-name soc-ai-tfstate-lock \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region ap-northeast-1
```

## 環境別デプロイ（Workspace）

```bash
# 本番環境
terraform workspace select prod || terraform workspace new prod

# ステージング環境
terraform workspace select staging || terraform workspace new staging
```

## 初回セットアップ

```bash
# 変数ファイルを作成
cp terraform.tfvars.example terraform.tfvars
# terraform.tfvars を編集して実際の値を設定

# 初期化
terraform init

# プラン確認
terraform plan

# 適用
terraform apply
```

## GitHub Actions の Secrets 設定

Terraform 適用後、以下を GitHub Secrets に登録してください：

| Secret 名 | 値 |
|---|---|
| `AWS_DEPLOY_ROLE_ARN` | `terraform output github_deploy_role_arn` の値 |
| `ECS_CLUSTER` | `terraform output ecs_cluster_name` の値 |
| `ECS_BACKEND_SERVICE` | `terraform output backend_service_name` の値 |
| `ECS_FRONTEND_SERVICE` | `terraform output frontend_service_name` の値 |
| `ECS_BACKEND_TASK_FAMILY` | `<project>-<env>-backend` |
| `ECS_FRONTEND_TASK_FAMILY` | `<project>-<env>-frontend` |

## リソース一覧

| リソース | 説明 |
|---|---|
| VPC | CIDR `10.0.0.0/16`、パブリック/プライベート各 2 AZ |
| ALB | パブリック。`/api/*` → バックエンド、それ以外 → フロントエンド |
| ECS クラスター | Fargate（本番）/ Fargate Spot（ステージング） |
| ECR | `soc-backend` / `soc-frontend`（最新 10 イメージ保持） |
| RDS | `create_rds=true` の場合のみ MySQL 8.0 を作成 |
| Secrets Manager | OpenAI API キー / DB クレデンシャルを参照 |
| CloudWatch Logs | `/ecs/soc-backend`・`/ecs/soc-frontend`（30 日保持） |
