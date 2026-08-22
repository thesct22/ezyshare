output "backend_url" {
  description = "Public HTTPS URL of the Cloud Run backend service"
  value       = google_cloud_run_v2_service.backend.uri
}

output "artifact_registry_url" {
  description = "URL of the Artifact Registry repository"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${var.artifact_repository_name}"
}
