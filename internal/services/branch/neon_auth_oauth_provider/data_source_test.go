package neon_auth_oauth_provider_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestNeonAuthOauthProviderDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/oauth_providers",
		testutil.JSONResponder(200, `{"providers": [`+oauthProviderJSON+`]}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_branch_neon_auth_oauth_provider" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  id         = "google"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_branch_neon_auth_oauth_provider.test", "id", "google"),
					testutil.CheckResourceAttr("data.neon_branch_neon_auth_oauth_provider.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("data.neon_branch_neon_auth_oauth_provider.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("data.neon_branch_neon_auth_oauth_provider.test", "type", "standard"),
					testutil.CheckResourceAttr("data.neon_branch_neon_auth_oauth_provider.test", "client_id", "my-client-id"),
					testutil.CheckResourceAttr("data.neon_branch_neon_auth_oauth_provider.test", "client_secret", "my-client-secret"),
				),
			},
		},
	})
}
