terraform {
  required_version = ">= 1.15"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.0"
    }
  }

  # Partial backend configuration (bucket passed via -backend-config at init)
  backend "gcs" {
    prefix = "backend/state"
  }
}