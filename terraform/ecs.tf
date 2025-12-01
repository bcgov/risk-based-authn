resource "aws_ecs_cluster" "rba_cluster" {
  name = "rba-cluster"
}

resource "aws_ecs_task_definition" "rba_task" {
  family                   = "rba-task"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "512"
  memory                   = "1024"
  container_definitions    = jsonencode([
    {
      name      = "rba"
      image     = "risk-based-authn"
      essential = true
      portMappings = [
        {
          containerPort = 8080
          protocol      = "tcp"
        }
      ]
    }
  ])
}

resource "aws_ecs_service" "rba_service" {
  name            = "rba-service"
  cluster         = aws_ecs_cluster.rba_cluster.id
  task_definition = aws_ecs_task_definition.rba_task.arn
  desired_count   = 1
  launch_type     = "FARGATE"
  network_configuration {
    subnets         = [aws_subnet.private1.id, aws_subnet.private2.id]
    security_groups = [aws_security_group.rba_sg.id]
  }
}
