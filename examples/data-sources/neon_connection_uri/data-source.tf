data "neon_connection_uri" "example" {
  project_id    = "your-project-id"
  database_name = "neondb"
  role_name     = "neondb_owner"
}
