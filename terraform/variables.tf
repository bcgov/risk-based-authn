variable "subnet_a" {
  type        = string
  description = "Value of the name tag for the app subnet in AZ a"
  default     = "Dev-App-A"
}

variable "subnet_b" {
  type        = string
  description = "Value of the name tag for the app subnet in AZ b"
  default     = "Dev-App-B"
}

variable "api_key_client" {
  type = string
  description = "The name of the client to use for hmac authentication"
}

variable "api_key_secret" {
  type = string
  description = "The name of the secret to use for hmac authentication"
}

variable "image_tag" {
  type    = string
  default = "latest"
}