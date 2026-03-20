terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # S3バックエンド（初回は backend.tfvars で設定）
  backend "s3" {
    bucket         = "soc-ai-tfstate"
    key            = "terraform.tfstate"
    region         = "ap-northeast-1"
    encrypt        = true
    dynamodb_table = "soc-ai-tfstate-lock"
  }
}

provider "aws" {
  region = var.aws_region
}

# ─── ローカル変数 ───────────────────────────────────────────
locals {
  env    = terraform.workspace # prod | staging
  prefix = "${var.project}-${local.env}"

  common_tags = {
    Project     = var.project
    Environment = local.env
    ManagedBy   = "terraform"
  }
}
