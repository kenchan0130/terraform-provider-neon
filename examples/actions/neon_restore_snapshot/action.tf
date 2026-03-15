action "neon_restore_snapshot" "example" {
  project_id       = neon_project.example.id
  snapshot_id      = "snap-example-id"
  name             = "restored-branch"
  finalize_restore = true
}
