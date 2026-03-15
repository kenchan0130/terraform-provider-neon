package masking_rules_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestMaskingRulesDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/masking_rules",
		testutil.JSONResponder(200, maskingRulesResponseJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_branch_masking_rules" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_branch_masking_rules.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("data.neon_branch_masking_rules.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("data.neon_branch_masking_rules.test", "masking_rules.#", "1"),
					testutil.CheckResourceAttr("data.neon_branch_masking_rules.test", "masking_rules.0.database_name", "mydb"),
					testutil.CheckResourceAttr("data.neon_branch_masking_rules.test", "masking_rules.0.schema_name", "public"),
					testutil.CheckResourceAttr("data.neon_branch_masking_rules.test", "masking_rules.0.table_name", "users"),
					testutil.CheckResourceAttr("data.neon_branch_masking_rules.test", "masking_rules.0.column_name", "email"),
					testutil.CheckResourceAttr("data.neon_branch_masking_rules.test", "masking_rules.0.masking_function", "anon.fake_email()"),
				),
			},
		},
	})
}
