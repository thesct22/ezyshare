output "backend_url" {
  description = "Public HTTPS URL of the Cloud Run backend service"
  value       = google_cloud_run_v2_service.backend.uri
}