variable "project_id" {
  description = "Google Cloud project ID"
  type        = string
}

variable "region" {
  description = "Google Cloud region"
  type        = string
  default     = "us-central1"
}

variable "service_name" {
  description = "Name of the Cloud Run service"
  type        = string
  default     = "ezyshare-backend"
}

variable "backend_container_image_name" {
  description = "Docker image on GHCR"
  type        = string
  default     = "ghcr.io/thesct22/ezyshare-backend:latest"
}

variable "cloud_run_ingress" {
  description = "Cloud Run ingress settings"
  type        = string
  default     = "INGRESS_TRAFFIC_ALL"
}

variable "backend_container_port" {
  description = "Backend container port"
  type        = number
  default     = 8080
}