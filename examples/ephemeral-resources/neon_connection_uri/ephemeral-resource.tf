ephemeral "neon_connection_uri" "example" {
  project_id    = neon_project.example.id
  database_name = "neondb"
  role_name     = "neondb_owner"
}
