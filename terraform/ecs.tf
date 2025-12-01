
##############################################
# ECS Cluster
##############################################
resource "aws_ecs_cluster" "main" {
  name = "rba-cluster"
}

##############################################
# RBA Task Definition
##############################################
resource "aws_ecs_task_definition" "rba" {
  family                   = "rba-task"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "512"
  memory                   = "1024"

  container_definitions = jsonencode([
    {
      name      = "rba"
      image     = "your-rba-image:latest"
      essential = true
      portMappings = [
        {
          containerPort = 8080
          protocol      = "tcp"
        }
      ]
      environment = [
        {
          name  = "REDIS_HOST"
          value = "redis" # Will resolve via Cloud Map
        },
        {
          name  = "REDIS_PORT"
          value = "6379"
        }
      ]
    }
  ])
}

##############################################
# RBA ECS Service
##############################################
resource "aws_ecs_service" "rba" {
  name            = "rba-service"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.rba.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = [data.aws_subnet.app_az_a.id, data.aws_subnet.app_az_b.id]
    security_groups = [aws_security_group.rba_sg.id]
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
  cpu                      = "512"
  memory                   = "1024"

  container_definitions = jsonencode([
    {
      name      = "redis"
      image     = "redis:7-alpine"
      essential = true
      portMappings = [
        {
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
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.redis.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = [data.aws_subnet.app_az_a.id, data.aws_subnet.app_az_b.id]
    security_groups = [aws_security_group.redis_sg.id]
    assign_public_ip = false
  }
}

