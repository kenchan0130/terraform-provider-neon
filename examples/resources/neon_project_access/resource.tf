resource "neon_project_access" "example" {
  project_id       = neon_project.example.id
  granted_to_email = "user@example.com"
}
