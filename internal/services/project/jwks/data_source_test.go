package jwks_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestJWKSDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/jwks",
		testutil.JSONResponder(200, `{"jwks": [`+jwksJSON+`]}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_project_jwks" "test" {
  project_id = "test-project-id"
  id         = "jwks-test-001"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_project_jwks.test", "id", "jwks-test-001"),
					testutil.CheckResourceAttr("data.neon_project_jwks.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("data.neon_project_jwks.test", "jwks_url", "https://example.com/.well-known/jwks.json"),
					testutil.CheckResourceAttr("data.neon_project_jwks.test", "provider_name", "Clerk"),
					testutil.CheckResourceAttr("data.neon_project_jwks.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("data.neon_project_jwks.test", "jwt_audience", "neon"),
				),
			},
		},
	})
}
