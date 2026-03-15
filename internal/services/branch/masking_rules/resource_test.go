package masking_rules_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const maskingRulesResponseJSON = `{
	"masking_rules": [
		{
			"database_name": "mydb",
			"schema_name": "public",
			"table_name": "users",
			"column_name": "email",
			"masking_function": "anon.fake_email()"
		}
	]
}`

func setupMaskingRulesMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPatch,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/masking_rules",
		testutil.JSONResponder(200, maskingRulesResponseJSON),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/masking_rules",
		testutil.JSONResponder(200, maskingRulesResponseJSON),
	)
}

func TestMaskingRulesResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupMaskingRulesMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_masking_rules" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"

  masking_rules {
    database_name    = "mydb"
    schema_name      = "public"
    table_name       = "users"
    column_name      = "email"
    masking_function = "anon.fake_email()"
  }
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_branch_masking_rules.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("neon_branch_masking_rules.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("neon_branch_masking_rules.test", "masking_rules.#", "1"),
					testutil.CheckResourceAttr("neon_branch_masking_rules.test", "masking_rules.0.database_name", "mydb"),
					testutil.CheckResourceAttr("neon_branch_masking_rules.test", "masking_rules.0.schema_name", "public"),
					testutil.CheckResourceAttr("neon_branch_masking_rules.test", "masking_rules.0.table_name", "users"),
					testutil.CheckResourceAttr("neon_branch_masking_rules.test", "masking_rules.0.column_name", "email"),
					testutil.CheckResourceAttr("neon_branch_masking_rules.test", "masking_rules.0.masking_function", "anon.fake_email()"),
				),
			},
		},
	})
}

func TestMaskingRulesResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupMaskingRulesMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_masking_rules" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"

  masking_rules {
    database_name    = "mydb"
    schema_name      = "public"
    table_name       = "users"
    column_name      = "email"
    masking_function = "anon.fake_email()"
  }
}
`),
			},
			{
				ResourceName:                         "neon_branch_masking_rules.test",
				ImportState:                          true,
				ImportStateId:                        "test-project-id/br-test-001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "project_id",
			},
		},
	})
}

func TestMaskingRulesResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPatch,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/masking_rules",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_masking_rules" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"

  masking_rules {
    database_name    = "mydb"
    schema_name      = "public"
    table_name       = "users"
    column_name      = "email"
    masking_function = "anon.fake_email()"
  }
}
`),
				ExpectError: regexp.MustCompile(`Failed to create masking rules`),
			},
		},
	})
}
