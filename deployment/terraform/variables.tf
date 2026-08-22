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

variable "artifact_repository_name" {
  description = "ID of the Artifact Registry repository for Docker images"
  type        = string
  default     = "ezyshare-repo"
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