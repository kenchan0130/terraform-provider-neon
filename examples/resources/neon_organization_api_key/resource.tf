resource "neon_organization_api_key" "example" {
  org_id = "org-my-organization-id"
  name   = "my-org-api-key"
}

# Optionally scope the key to a specific project
resource "neon_organization_api_key" "project_scoped" {
  org_id     = "org-my-organization-id"
  name       = "my-project-scoped-key"
  project_id = "my-project-id"
}
