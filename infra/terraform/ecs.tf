# ─── ECS クラスター ───────────────────────────────────────────
resource "aws_ecs_cluster" "main" {
  name = "${local.prefix}-cluster"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }

  tags = merge(local.common_tags, { Name = "${local.prefix}-cluster" })
}

resource "aws_ecs_cluster_capacity_providers" "main" {
  cluster_name       = aws_ecs_cluster.main.name
  capacity_providers = ["FARGATE", "FARGATE_SPOT"]

  default_capacity_provider_strategy {
    capacity_provider = local.env == "prod" ? "FARGATE" : "FARGATE_SPOT"
    weight            = 1
  }
}

# ─── ECS タスク定義: バックエンド ─────────────────────────────
resource "aws_ecs_task_definition" "backend" {
  family                   = "${local.prefix}-backend"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = tostring(var.backend_cpu)
  memory                   = tostring(var.backend_memory)
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name      = "soc-backend"
      image     = "${aws_ecr_repository.backend.repository_url}:${var.backend_image_tag}"
      essential = true

      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
          protocol      = "tcp"
        }
      ]

      environment = [
        { name = "APP_ENV", value = local.env },
        { name = "SERVER_PORT", value = "8080" },
        { name = "DB_HOST", value = local.db_host },
        { name = "DB_PORT", value = "3306" },
        { name = "DB_NAME", value = var.db_name },
        # DB_USER は Secrets Manager から取得しない場合のフォールバック
        { name = "DB_USER", value = var.db_username },
        { name = "BACKEND_URL", value = "http://localhost:8080" }
      ]

      # secrets: Secrets Manager の JSON キーを個別環境変数に展開
      # RDS マネージドシークレット形式: {"username":"...","password":"...","host":"...","port":...}
      # valueFrom の末尾 ":json-key::" で特定フィールドのみ取得できる
      secrets = concat(
        var.openai_secret_arn != "" ? [
          {
            name      = "OPENAI_API_KEY"
            valueFrom = var.openai_secret_arn
          }
        ] : [],
        var.db_secret_arn != "" ? [
          {
            # config.go が読む "DB_PASSWORD" に直接マッピング
            name      = "DB_PASSWORD"
            valueFrom = "${var.db_secret_arn}:password::"
          },
          {
            # DB_USER も Secrets Manager から上書き（オプション）
            name      = "DB_USER"
            valueFrom = "${var.db_secret_arn}:username::"
          }
        ] : []
      )

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.backend.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs"
        }
      }

      healthCheck = {
        command     = ["CMD-SHELL", "wget -qO- http://localhost:8080/health || exit 1"]
        interval    = 30
        timeout     = 5
        retries     = 3
        startPeriod = 60
      }
    }
  ])

  tags = local.common_tags
}

# ─── ECS サービス: バックエンド ───────────────────────────────
resource "aws_ecs_service" "backend" {
  name            = "${local.prefix}-backend"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.backend.arn
  desired_count   = var.backend_desired_count

  launch_type = local.env == "prod" ? "FARGATE" : null

  dynamic "capacity_provider_strategy" {
    for_each = local.env != "prod" ? [1] : []
    content {
      capacity_provider = "FARGATE_SPOT"
      weight            = 1
    }
  }

  network_configuration {
    subnets          = aws_subnet.private[*].id
    security_groups  = [aws_security_group.backend.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.backend.arn
    container_name   = "soc-backend"
    container_port   = 8080
  }

  deployment_minimum_healthy_percent = 50
  deployment_maximum_percent         = 200

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  depends_on = [
    aws_lb_listener.http,
    aws_iam_role_policy_attachment.ecs_task_execution
  ]

  tags = local.common_tags

  lifecycle {
    # CI/CD でイメージタグが変わるため ignore_changes で管理
    ignore_changes = [task_definition, desired_count]
  }
}

# ─── ECS タスク定義: フロントエンド ───────────────────────────
resource "aws_ecs_task_definition" "frontend" {
  family                   = "${local.prefix}-frontend"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = tostring(var.frontend_cpu)
  memory                   = tostring(var.frontend_memory)
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name      = "soc-frontend"
      image     = "${aws_ecr_repository.frontend.repository_url}:${var.frontend_image_tag}"
      essential = true

      portMappings = [
        {
          containerPort = 3000
          hostPort      = 3000
          protocol      = "tcp"
        }
      ]

      environment = [
        { name = "APP_ENV", value = local.env },
        { name = "NODE_ENV", value = local.env == "prod" ? "production" : "development" },
        # ECS 内部通信: バックエンドサービス名（service discovery or ALB 経由）
        { name = "BACKEND_URL", value = "http://${local.prefix}-backend.${local.prefix}-cluster:8080" },
        { name = "NEXT_PUBLIC_BACKEND_URL", value = "http://localhost:8080" }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.frontend.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs"
        }
      }
    }
  ])

  tags = local.common_tags
}

# ─── ECS サービス: フロントエンド ─────────────────────────────
resource "aws_ecs_service" "frontend" {
  name            = "${local.prefix}-frontend"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.frontend.arn
  desired_count   = var.frontend_desired_count

  launch_type = local.env == "prod" ? "FARGATE" : null

  dynamic "capacity_provider_strategy" {
    for_each = local.env != "prod" ? [1] : []
    content {
      capacity_provider = "FARGATE_SPOT"
      weight            = 1
    }
  }

  network_configuration {
    subnets          = aws_subnet.private[*].id
    security_groups  = [aws_security_group.frontend.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.frontend.arn
    container_name   = "soc-frontend"
    container_port   = 3000
  }

  deployment_minimum_healthy_percent = 50
  deployment_maximum_percent         = 200

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  depends_on = [
    aws_lb_listener.http,
    aws_iam_role_policy_attachment.ecs_task_execution
  ]

  tags = local.common_tags

  lifecycle {
    ignore_changes = [task_definition, desired_count]
  }
}
