##############################################
# ECS Cluster
##############################################
resource "aws_ecs_cluster" "rba" {
  name = "rba-cluster"
}

##############################################
# RBA Task Definition
##############################################
resource "aws_ecs_task_definition" "rba" {
  family                   = "rba-task"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = aws_iam_role.rba_task_execution_role.arn
  task_role_arn            = aws_iam_role.rba_task_role.arn

  container_definitions = jsonencode([
    {
      name      = "rba"
      image     = "ghcr.io/bcgov/risk-based-authn/risk-based-authn:${var.image_tag}"
      essential = true
      portMappings = [
        {
          name          = "rba"
          containerPort = 8080
          protocol      = "tcp"
        }
      ]
      environment = [
        {
          name  = "API_KEY"
          value = var.api_key_client 
        },
        {
          name  = "API_SECRET"
          value = var.api_key_secret
        },
        {
          name  = "PORT"
          value = "8080"
        }
      ]
      logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-create-group  = "true"
        awslogs-group         = "rba-logs"
        awslogs-region        = "ca-central-1"
        awslogs-stream-prefix = "ecs"
      }
      }
    }
  ])
}

##############################################
# RBA ECS Service
##############################################
resource "aws_ecs_service" "rba" {
  name            = "rba-service"
  cluster         = aws_ecs_cluster.rba.id
  task_definition = aws_ecs_task_definition.rba.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  service_connect_configuration {
    enabled = true
    namespace = aws_service_discovery_private_dns_namespace.this.arn

    service {
      port_name = "rba"
      client_alias {
        dns_name = "rba"
        port     = 8080
      }
    }
  }

  network_configuration {
    subnets         = [data.aws_subnet.app_az_a.id, data.aws_subnet.app_az_b.id]
    security_groups = [data.aws_security_group.app_sg.id]
    assign_public_ip = false
  }
}

##############################################
# Redis Task Definition
##############################################
resource "aws_ecs_task_definition" "redis" {
  family                   = "redis-task"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"

  container_definitions = jsonencode([
    {
      name      = "redis"
      image     = "redis:7-alpine"
      essential = true
      portMappings = [
        {
          name          = "redis"
          containerPort = 6379
          protocol      = "tcp"
        }
      ]
    }
  ])
}

##############################################
# Redis ECS Service
##############################################
resource "aws_ecs_service" "redis" {
  name            = "redis-service"
  cluster         = aws_ecs_cluster.rba.id
  task_definition = aws_ecs_task_definition.redis.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  service_connect_configuration {
    enabled = true

    namespace = aws_service_discovery_private_dns_namespace.this.arn
    service {
      port_name = "redis"
      client_alias {
        dns_name = "redis"
        port     = 6379
      }
    }
  }

  network_configuration {
    subnets         = [data.aws_subnet.app_az_a.id, data.aws_subnet.app_az_b.id]
    security_groups = [data.aws_security_group.app_sg.id]
    assign_public_ip = false
  }
}

################################################################################
# Service Discovery
################################################################################
resource "aws_service_discovery_private_dns_namespace" "this" {
  name        = "rba.local"
  description = "Private DNS namespace for RBA"
  vpc         = data.aws_vpc.main.id
}
