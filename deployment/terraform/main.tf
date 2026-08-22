locals {
  runtime_sa_email = "sa-ezyshare-backend@${var.project_id}.iam.gserviceaccount.com"
  image_url        = "${var.region}-docker.pkg.dev/${var.project_id}/${var.artifact_repository_name}/backend:latest"
}

# Artifact Registry Repository with 24h Auto-Cleanup Policy for untagged images
resource "google_artifact_registry_repository" "backend_repo" {
  location      = var.region
  repository_id = var.artifact_repository_name
  description   = "Docker repository for EzyShare containers"
  format        = "DOCKER"

  cleanup_policies {
    id     = "delete-untagged-images"
    action = "DELETE"
    condition {
      tag_state  = "UNTAGGED"
      older_than = "86400s" # 1 day
    }
  }
}

# Cloud Run Service (v2 API)
resource "google_cloud_run_v2_service" "backend" {
  name                = var.service_name
  location            = var.region
  ingress             = var.cloud_run_ingress
  deletion_protection = false

  depends_on = [google_artifact_registry_repository.backend_repo]

  template {
    service_account = local.runtime_sa_email

    containers {
      image = "gcr.io/cloudrun/hello" # Starter placeholder so initial terraform apply succeeds before docker push
      ports {
        container_port = var.backend_container_port
      }

      env {
        name  = "APP_ENV"
        value = "prod"
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [
      client,
      client_version,
      template[0].containers[0].image
    ]
  }
}

# Allow Public Access to Cloud Run Service
resource "google_cloud_run_v2_service_iam_member" "public_access" {
  name     = google_cloud_run_v2_service.backend.name
  location = google_cloud_run_v2_service.backend.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
