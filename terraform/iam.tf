data "aws_iam_policy_document" "this" {
  version = "2012-10-17"
  statement {
    sid     = ""
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "rba_task_execution_role" {
  name               = "RBAProviderTaskExecutionRole"
  assume_role_policy = data.aws_iam_policy_document.this.json
}

resource "aws_iam_role_policy_attachment" "task_execution_role_policy_attachment" {
  role       = aws_iam_role.rba_task_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role" "rba_task_role" {
  name = "RBAProviderTaskRole"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ecs-tasks.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "rba_task_role_s3" {
  name   = "RBAProviderTaskRoleS3Policy"
  role   = aws_iam_role.rba_task_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = [
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = [
          "arn:aws:s3:::${var.geoip_s3_bucket}",
          "arn:aws:s3:::${var.geoip_s3_bucket}/*"
        ]
      }
    ]
  })
}


resource "aws_iam_role_policy" "rba_task_execution_cwlogs" {
  name = "RBAProviderLogsPolicy"
  role = aws_iam_role.rba_task_execution_role.id

  policy = <<-EOF
  {
      "Version": "2012-10-17",
      "Statement": [
          {
              "Effect": "Allow",
              "Action": [
                  "logs:CreateLogGroup",
                  "logs:CreateLogStream",
                  "logs:PutLogEvents",
                  "logs:DescribeLogStreams"
              ],
              "Resource": [
                  "arn:aws:logs:*:*:*"
              ]
          }
      ]
  }
  EOF
}