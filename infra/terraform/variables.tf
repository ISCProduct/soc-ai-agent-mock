# ─── 共通 ────────────────────────────────────────────────────
variable "aws_region" {
  description = "AWSリージョン"
  type        = string
  default     = "ap-northeast-1"
}

variable "project" {
  description = "プロジェクト名（リソース名のプレフィックス）"
  type        = string
  default     = "soc-ai"
}

# ─── ネットワーク ────────────────────────────────────────────
variable "vpc_cidr" {
  description = "VPC CIDR ブロック"
  type        = string
  default     = "10.0.0.0/16"
}

variable "public_subnet_cidrs" {
  description = "パブリックサブネット CIDR（AZ分）"
  type        = list(string)
  default     = ["10.0.1.0/24", "10.0.2.0/24"]
}

variable "private_subnet_cidrs" {
  description = "プライベートサブネット CIDR（AZ分）"
  type        = list(string)
  default     = ["10.0.11.0/24", "10.0.12.0/24"]
}

variable "availability_zones" {
  description = "使用するアベイラビリティゾーン"
  type        = list(string)
  default     = ["ap-northeast-1a", "ap-northeast-1c"]
}

# ─── ECS ────────────────────────────────────────────────────
variable "backend_image_tag" {
  description = "バックエンドコンテナのイメージタグ"
  type        = string
  default     = "latest"
}

variable "frontend_image_tag" {
  description = "フロントエンドコンテナのイメージタグ"
  type        = string
  default     = "latest"
}

variable "backend_cpu" {
  description = "バックエンドタスクの CPU ユニット"
  type        = number
  default     = 2048
}

variable "backend_memory" {
  description = "バックエンドタスクのメモリ (MiB)"
  type        = number
  default     = 8192
}

variable "backend_desired_count" {
  description = "バックエンドサービスの希望タスク数"
  type        = number
  default     = 1
}

variable "frontend_cpu" {
  description = "フロントエンドタスクの CPU ユニット"
  type        = number
  default     = 512
}

variable "frontend_memory" {
  description = "フロントエンドタスクのメモリ (MiB)"
  type        = number
  default     = 1024
}

variable "frontend_desired_count" {
  description = "フロントエンドサービスの希望タスク数"
  type        = number
  default     = 1
}

# ─── RDS ────────────────────────────────────────────────────
variable "create_rds" {
  description = "RDSを新規作成するか（falseの場合は既存DBエンドポイントを使用）"
  type        = bool
  default     = false
}

variable "existing_rds_endpoint" {
  description = "既存RDSのエンドポイント（create_rds=falseの場合に使用）"
  type        = string
  default     = ""
}

variable "db_name" {
  description = "データベース名"
  type        = string
  default     = "app_db"
}

variable "db_username" {
  description = "DBマスターユーザー名"
  type        = string
  default     = "app_user"
}

variable "db_instance_class" {
  description = "RDB インスタンスクラス"
  type        = string
  default     = "db.t3.small"
}

variable "db_allocated_storage" {
  description = "RDB ストレージ容量 (GB)"
  type        = number
  default     = 20
}

# ─── Secrets Manager ────────────────────────────────────────
variable "openai_secret_arn" {
  description = "OpenAI APIキーが格納されたSecrets ManagerのARN"
  type        = string
  default     = ""
}

variable "db_secret_arn" {
  description = "DBクレデンシャルが格納されたSecrets ManagerのARN"
  type        = string
  default     = ""
}

# ─── ALB / ドメイン ──────────────────────────────────────────
variable "certificate_arn" {
  description = "ACM 証明書の ARN（HTTPS 用。空の場合は HTTP のみ）"
  type        = string
  default     = ""
}

variable "backend_health_check_path" {
  description = "バックエンドのヘルスチェックパス"
  type        = string
  default     = "/health"
}
