data "neon_branches" "example" {
  project_id = "your-project-id"

  query = {
    search     = "main"
    sort_by    = "name"
    sort_order = "asc"
  }
}
