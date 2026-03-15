resource "neon_project_jwks" "example" {
  project_id    = neon_project.example.id
  jwks_url      = "https://example.com/.well-known/jwks.json"
  provider_name = "Clerk"
  branch_id     = neon_branch.example.id
  jwt_audience  = "neon"
}
