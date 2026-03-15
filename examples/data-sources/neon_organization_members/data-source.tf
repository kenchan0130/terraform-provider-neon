data "neon_organization_members" "example" {
  org_id = "your-organization-id"

  query = {
    sort_by    = "email"
    sort_order = "asc"
  }
}
