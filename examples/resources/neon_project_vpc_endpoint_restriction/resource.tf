resource "neon_project_vpc_endpoint_restriction" "example" {
  project_id      = neon_project.example.id
  vpc_endpoint_id = neon_organization_vpc_endpoint.example.vpc_endpoint_id
  label           = "my-vpc-endpoint"
}
