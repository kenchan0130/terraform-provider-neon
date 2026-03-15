resource "neon_anonymized_branch" "example" {
  project_id         = neon_project.example.id
  name               = "anonymized-branch"
  start_anonymization = true

  masking_rules {
    database_name    = "mydb"
    schema_name      = "public"
    table_name       = "users"
    column_name      = "email"
    masking_function = "anon.fake_email()"
  }

  masking_rules {
    database_name = "mydb"
    schema_name   = "public"
    table_name    = "users"
    column_name   = "name"
    masking_value = "'REDACTED'"
  }
}
