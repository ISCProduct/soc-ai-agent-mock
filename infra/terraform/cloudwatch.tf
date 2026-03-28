# ─── CloudWatch Log Groups ────────────────────────────────────
resource "aws_cloudwatch_log_group" "backend" {
  name              = "/ecs/soc-backend"
  retention_in_days = 30
  tags              = local.common_tags
}

resource "aws_cloudwatch_log_group" "frontend" {
  name              = "/ecs/soc-frontend"
  retention_in_days = 30
  tags              = local.common_tags
}
