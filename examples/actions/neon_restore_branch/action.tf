action "neon_restore_branch" "example" {
  project_id         = neon_project.example.id
  branch_id          = neon_branch.example.id
  source_branch_id   = neon_branch.source.id
  preserve_under_name = "backup-before-restore"
}
