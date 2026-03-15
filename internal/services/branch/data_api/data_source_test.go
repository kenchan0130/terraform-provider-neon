package data_api_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestBranchDataAPIDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/data-api/test-db",
		testutil.JSONResponder(200, dataAPIGetResponseJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_branch_data_api" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  database_name = "test-db"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_branch_data_api.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("data.neon_branch_data_api.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("data.neon_branch_data_api.test", "database_name", "test-db"),
					testutil.CheckResourceAttr("data.neon_branch_data_api.test", "url", "https://data-api.example.com/test-db"),
					testutil.CheckResourceAttr("data.neon_branch_data_api.test", "status", "active"),
				),
			},
		},
	})
}
