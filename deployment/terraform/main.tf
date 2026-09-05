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

    # RoomManager keeps all room/peer state in an in-process Go map with no
    # shared backing store. Cloud Run gives no guarantee that two different
    # clients' WebSocket connections land on the same instance, so allowing
    # more than one instance means a room created on instance A is invisible
    # to a peer routed to instance B ("room not found"). Pin to exactly one
    # instance until room state is moved to a shared store (see
    # docs/superpowers/plans/2026-09-06-horizontal-scaling-plan.md).
    scaling {
      max_instance_count = 1
    }

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
