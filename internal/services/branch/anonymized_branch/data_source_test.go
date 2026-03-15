package anonymized_branch_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestAnonymizedBranchDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-anon-001",
		testutil.JSONResponder(200, `{"branch": `+branchJSON+`, "annotation": {"object": {"type": "branch", "id": "br-anon-001"}, "value": {}}}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-anon-001/masking_rules",
		testutil.JSONResponder(200, maskingRulesJSON),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-anon-001/anonymized_status",
		testutil.JSONResponder(200, anonymizedStatusJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_anonymized_branch" "test" {
  project_id = "test-project-id"
  id         = "br-anon-001"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_anonymized_branch.test", "id", "br-anon-001"),
					testutil.CheckResourceAttr("data.neon_anonymized_branch.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("data.neon_anonymized_branch.test", "name", "anon-branch"),
					testutil.CheckResourceAttr("data.neon_anonymized_branch.test", "parent_id", "br-parent-001"),
					testutil.CheckResourceAttr("data.neon_anonymized_branch.test", "state", "created"),
					testutil.CheckResourceAttr("data.neon_anonymized_branch.test", "masking_rules.#", "1"),
					testutil.CheckResourceAttr("data.neon_anonymized_branch.test", "masking_rules.0.database_name", "mydb"),
					testutil.CheckResourceAttr("data.neon_anonymized_branch.test", "masking_rules.0.masking_function", "anon.fake_email()"),
				),
			},
		},
	})
}
