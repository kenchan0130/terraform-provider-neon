resource "neon_database" "example" {
  project_id = neon_project.example.id
  branch_id  = neon_branch.example.id
  name       = "mydb"
  owner_name = neon_role.example.name
}
