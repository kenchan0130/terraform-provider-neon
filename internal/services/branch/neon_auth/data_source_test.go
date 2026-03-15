package neon_auth_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestNeonAuthDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth",
		testutil.JSONResponder(200, neonAuthIntegrationJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_branch_neon_auth" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_branch_neon_auth.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("data.neon_branch_neon_auth.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("data.neon_branch_neon_auth.test", "auth_provider", "stack_v2"),
					testutil.CheckResourceAttr("data.neon_branch_neon_auth.test", "auth_provider_project_id", "auth-proj-001"),
					testutil.CheckResourceAttr("data.neon_branch_neon_auth.test", "db_name", "neondb"),
					testutil.CheckResourceAttr("data.neon_branch_neon_auth.test", "jwks_url", "https://example.com/.well-known/jwks.json"),
					testutil.CheckResourceAttr("data.neon_branch_neon_auth.test", "base_url", "https://auth.example.com"),
				),
			},
		},
	})
}
