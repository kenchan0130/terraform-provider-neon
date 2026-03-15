resource "neon_branch_neon_auth_oauth_provider" "example" {
  project_id    = "your-project-id"
  branch_id     = "your-branch-id"
  type          = "standard"
  client_id     = "your-oauth-client-id"
  client_secret = "your-oauth-client-secret"
}
