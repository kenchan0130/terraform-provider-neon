resource "neon_role" "example" {
  project_id = neon_project.example.id
  branch_id  = neon_branch.example.id
  name       = "myrole"
}
