# ─── ネットワーク ────────────────────────────────────────────
output "vpc_id" {
  description = "VPC ID"
  value       = aws_vpc.main.id
}

output "public_subnet_ids" {
  description = "パブリックサブネット ID 一覧"
  value       = aws_subnet.public[*].id
}

output "private_subnet_ids" {
  description = "プライベートサブネット ID 一覧"
  value       = aws_subnet.private[*].id
}

# ─── ALB ─────────────────────────────────────────────────────
output "alb_dns_name" {
  description = "ALB の DNS 名"
  value       = aws_lb.main.dns_name
}

output "alb_zone_id" {
  description = "ALB の Route53 ゾーン ID"
  value       = aws_lb.main.zone_id
}

output "app_url" {
  description = "アプリケーション URL（ALB 経由）"
  value       = "http://${aws_lb.main.dns_name}"
}

# ─── ECR ─────────────────────────────────────────────────────
output "ecr_backend_url" {
  description = "バックエンド ECR リポジトリ URL"
  value       = aws_ecr_repository.backend.repository_url
}

output "ecr_frontend_url" {
  description = "フロントエンド ECR リポジトリ URL"
  value       = aws_ecr_repository.frontend.repository_url
}

# ─── ECS ─────────────────────────────────────────────────────
output "ecs_cluster_name" {
  description = "ECS クラスター名"
  value       = aws_ecs_cluster.main.name
}

output "backend_service_name" {
  description = "バックエンド ECS サービス名"
  value       = aws_ecs_service.backend.name
}

output "frontend_service_name" {
  description = "フロントエンド ECS サービス名"
  value       = aws_ecs_service.frontend.name
}

# ─── IAM ─────────────────────────────────────────────────────
output "ecs_task_execution_role_arn" {
  description = "ECS タスク実行ロール ARN"
  value       = aws_iam_role.ecs_task_execution.arn
}

output "github_deploy_role_arn" {
  description = "GitHub Actions デプロイロール ARN"
  value       = aws_iam_role.github_deploy.arn
}

# ─── RDS ─────────────────────────────────────────────────────
output "db_endpoint" {
  description = "DB エンドポイント"
  value       = local.db_host
}
