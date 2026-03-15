resource "neon_snapshot_schedule" "example" {
  project_id = "your-project-id"
  branch_id  = "your-branch-id"

  schedule {
    frequency         = "daily"
    hour              = 3
    retention_seconds = 86400
  }
}
