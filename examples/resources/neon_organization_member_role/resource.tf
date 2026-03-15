resource "neon_organization_member_role" "example" {
  org_id = "your-organization-id"
  member_id       = "12345678-1234-1234-1234-123456789012"
  role            = "member"
}
