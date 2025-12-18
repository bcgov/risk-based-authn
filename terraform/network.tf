##############################################
# Networking - Using Existing LZA Resources #
##############################################

# Fetch the existing VPC
data "aws_vpc" "main" {
  state = "available"
}

# Fetch App subnets in both AZs
data "aws_subnet" "app_az_a" {
  filter {
    name   = "tag:Name"
    values = [var.subnet_a]
  }
}

data "aws_subnet" "app_az_b" {
  filter {
    name   = "tag:Name"
    values = [var.subnet_b]
  }
}

data "aws_security_group" "app_sg" {
  filter {
    name   = "tag:Name"
    values = ["App"]
  }

  vpc_id = data.aws_vpc.main.id
}
