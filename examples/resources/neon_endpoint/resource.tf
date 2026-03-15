resource "neon_endpoint" "example" {
  project_id             = neon_project.example.id
  branch_id              = neon_branch.example.id
  type                   = "read_write"
  autoscaling_limit_min_cu = 0.25
  autoscaling_limit_max_cu = 1
  suspend_timeout_seconds  = 300
}
