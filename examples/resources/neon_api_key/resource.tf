resource "neon_api_key" "example" {
  name = "my-api-key"
}

output "api_key" {
  value     = neon_api_key.example.key
  sensitive = true
}
