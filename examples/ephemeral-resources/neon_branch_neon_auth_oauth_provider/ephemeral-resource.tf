ephemeral "neon_branch_neon_auth_oauth_provider" "example" {
  project_id = neon_project.example.id
  branch_id  = neon_branch.example.id
  id         = "google"
}
