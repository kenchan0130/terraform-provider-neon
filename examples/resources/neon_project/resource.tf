resource "neon_project" "example" {
  name                    = "my-project"
  region_id               = "aws-us-east-1"
  pg_version              = 16
  history_retention_seconds = 86400
  store_passwords         = true
}
