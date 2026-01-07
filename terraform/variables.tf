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

variable "jwt_aud" {
  type = string
  description = "The expected audience for service account"
}

variable "jwks_url" {
  type = string
  description = "The jwks url for the auth server"
}

variable "image_tag" {
  type    = string
  default = "latest"
}