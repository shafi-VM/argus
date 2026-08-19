variable "region" {
  description = "AWS region to deploy into."
  type        = string
  default     = "us-east-1"
}

variable "instance_type" {
  description = "EC2 instance type. SigNoz (ClickHouse) wants ~8 GB RAM, so t3.large is the comfortable floor for the full LEARN tier. For a PREVENT-only box you can drop to t3.medium (4 GB)."
  type        = string
  default     = "t3.large"
}

variable "allowed_cidr" {
  description = "CIDR allowed to reach SSH (22), the SigNoz UI (8081) and Argus (8088). Scope this to YOUR IP: `echo \"$(curl -s ifconfig.me)/32\"`. Do NOT use 0.0.0.0/0 — the Argus proxy forwards upstream credentials and SigNoz would be world-readable."
  type        = string

  validation {
    condition     = var.allowed_cidr != "0.0.0.0/0"
    error_message = "Refusing 0.0.0.0/0: scope allowed_cidr to your own IP (e.g. 203.0.113.4/32)."
  }
}

variable "key_name" {
  description = "Name of an EXISTING EC2 key pair for SSH. Create one with `aws ec2 create-key-pair --key-name argus --query KeyMaterial --output text > argus.pem && chmod 400 argus.pem`."
  type        = string
}

variable "root_volume_gb" {
  description = "Root EBS (gp3) size in GB. ClickHouse stores telemetry here; 40 GB is a comfortable demo floor."
  type        = number
  default     = 40
}

variable "repo_url" {
  description = "Argus repository to clone and run on the box."
  type        = string
  default     = "https://github.com/shafi-VM/argus.git"
}

variable "project" {
  description = "Name tag / prefix for created resources."
  type        = string
  default     = "argus"
}
