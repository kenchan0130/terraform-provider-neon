data "neon_projects" "example" {
  query = {
    search = "my-project"
  }
}

data "neon_projects" "by_org" {
  query = {
    org_id = "org-example-001"
  }
}

data "neon_projects" "recoverable" {
  query = {
    recoverable = true
  }
}
