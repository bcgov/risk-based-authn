##############################################
# Networking - Using Existing LZA Resources #
##############################################

# Fetch the existing VPC
data "aws_vpc" "main" {
  tags = {
    Accelerator = "true"
  }
}

# Fetch App subnets in both AZs
data "aws_subnet" "app_az_a" {
  tags = {
    Tier = "App"
    AZ   = "A"
  }
}

data "aws_subnet" "app_az_b" {
  tags = {
    Tier = "App"
    AZ   = "B"
  }
}

# Optional: Fetch Web subnets if needed for ALB
data "aws_subnet" "web_az_a" {
  tags = {
    Tier = "Web"
    AZ   = "A"
  }
}

data "aws_subnet" "web_az_b" {
  tags = {
    Tier = "Web"
    AZ   = "B"
  }
}

# Security Group for RBA workload
resource "aws_security_group" "rba_sg" {
  name        = "rba-sg"
  description = "Security group for RBA workload"
  vpc_id      = data.aws_vpc.main.id

  ingress {
    description = "Allow app traffic on port 8080 from within VPC"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.main.cidr_block]
  }

  egress {
    description = "Allow all outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "rba-sg"
  }
}

# Security Group for Redis
resource "aws_security_group" "redis_sg" {
  name        = "redis-sg"
  description = "Security group for Redis"
  vpc_id      = data.aws_vpc.main.id

  ingress {
    description     = "Allow Redis traffic from RBA SG"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [aws_security_group.rba_sg.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "redis-sg"
  }
}

##############################################
# Outputs for convenience
##############################################
output "vpc_id" {
  value = data.aws_vpc.main.id
}

output "app_subnets" {
  value = [data.aws_subnet.app_az_a.id, data.aws_subnet.app_az_b.id]
}

output "web_subnets" {
  value = [data.aws_subnet.web_az_a.id, data.aws_subnet.web_az_b.id]
}
