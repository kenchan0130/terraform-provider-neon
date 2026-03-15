action "neon_role_password_reset" "example" {
  project_id = neon_project.example.id
  branch_id  = neon_branch.example.id
  role_name  = neon_role.example.name
}
