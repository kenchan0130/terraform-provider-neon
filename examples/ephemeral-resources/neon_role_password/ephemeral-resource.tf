ephemeral "neon_role_password" "example" {
  project_id = neon_project.example.id
  branch_id  = neon_branch.example.id
  role_name  = neon_role.example.name
}
