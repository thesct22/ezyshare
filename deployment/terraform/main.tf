locals {
  runtime_sa_email = "sa-ezyshare-backend@${var.project_id}.iam.gserviceaccount.com"
}

# Cloud Run Service (v2 API)
resource "google_cloud_run_v2_service" "backend" {
  name     = var.service_name
  location = var.region
  ingress  = var.cloud_run_ingress

  template {
    service_account = local.runtime_sa_email

    containers {
      image = var.backend_container_image_name
      ports {
        container_port = var.backend_container_port
      }

      env {
        name  = "PORT"
        value = tostring(var.backend_container_port)
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