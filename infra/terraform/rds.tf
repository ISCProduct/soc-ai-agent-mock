# ─── RDS サブネットグループ ───────────────────────────────────
resource "aws_db_subnet_group" "main" {
  count      = var.create_rds ? 1 : 0
  name       = "${local.prefix}-db-subnet"
  subnet_ids = aws_subnet.private[*].id
  tags       = merge(local.common_tags, { Name = "${local.prefix}-db-subnet" })
}

# ─── RDS MySQL 8.0 ───────────────────────────────────────────
resource "aws_db_instance" "main" {
  count = var.create_rds ? 1 : 0

  identifier     = "${local.prefix}-mysql"
  engine         = "mysql"
  engine_version = "8.0"
  instance_class = var.db_instance_class

  allocated_storage     = var.db_allocated_storage
  max_allocated_storage = var.db_allocated_storage * 2
  storage_type          = "gp3"
  storage_encrypted     = true

  db_name  = var.db_name
  username = var.db_username
  # パスワードは Secrets Manager から取得するため manage_master_user_password を使用
  manage_master_user_password = true

  db_subnet_group_name   = aws_db_subnet_group.main[0].name
  vpc_security_group_ids = [aws_security_group.rds.id]

  multi_az               = local.env == "prod"
  publicly_accessible    = false
  deletion_protection    = local.env == "prod"
  skip_final_snapshot    = local.env != "prod"
  final_snapshot_identifier = local.env == "prod" ? "${local.prefix}-final-snapshot" : null

  backup_retention_period = local.env == "prod" ? 7 : 1
  backup_window           = "03:00-04:00"
  maintenance_window      = "Mon:04:00-Mon:05:00"

  parameter_group_name = aws_db_parameter_group.main[0].name

  tags = merge(local.common_tags, { Name = "${local.prefix}-mysql" })
}

resource "aws_db_parameter_group" "main" {
  count  = var.create_rds ? 1 : 0
  name   = "${local.prefix}-mysql8"
  family = "mysql8.0"

  parameter {
    name  = "character_set_server"
    value = "utf8mb4"
  }

  parameter {
    name  = "collation_server"
    value = "utf8mb4_unicode_ci"
  }

  parameter {
    name  = "time_zone"
    value = "Asia/Tokyo"
  }

  tags = local.common_tags
}

# ─── ローカル: DB エンドポイント解決 ─────────────────────────
# create_rds=true なら作成したRDSを使い、falseなら変数から取得
locals {
  db_host = var.create_rds ? aws_db_instance.main[0].address : var.existing_rds_endpoint
}
