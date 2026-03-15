resource "neon_branch" "example" {
  project_id = neon_project.example.id
  name       = "dev-branch"
}
