resource "neon_branch_masking_rules" "example" {
  project_id = "my-project-id"
  branch_id  = "br-example-001"

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
