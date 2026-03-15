terraform {
  required_providers {
    neon = {
      source = "kenchan0130/neon"
    }
  }
}

provider "neon" {
  # api_key = "..." # Or set NEON_API_KEY environment variable
}
