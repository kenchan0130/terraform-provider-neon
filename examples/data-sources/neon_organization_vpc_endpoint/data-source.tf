data "neon_organization_vpc_endpoint" "example" {
  org_id          = "your-organization-id"
  region_id       = "aws-us-east-1"
  vpc_endpoint_id = "vpce-abc123"
}
