action "neon_set_default_branch" "example" {
  project_id = neon_project.example.id
  branch_id  = neon_branch.example.id
}
