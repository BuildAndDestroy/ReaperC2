variable "do_token" {
  description = "DigitalOcean API Token"
  type        = string
  sensitive   = true
}

variable "region" {
  description = "Region for deployment"
  type        = string
  default     = "nyc3"
}
