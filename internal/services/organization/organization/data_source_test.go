package organization_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const organizationJSON = `{
	"id": "org-test-001",
	"name": "My Organization",
	"handle": "my-org",
	"plan": "business",
	"managed_by": "console",
	"allow_hipaa_projects": false,
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z"
}`

func TestOrganizationDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/organizations/org-test-001",
		testutil.JSONResponder(200, organizationJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_organization" "test" {
  id = "org-test-001"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_organization.test", "id", "org-test-001"),
					testutil.CheckResourceAttr("data.neon_organization.test", "name", "My Organization"),
					testutil.CheckResourceAttr("data.neon_organization.test", "handle", "my-org"),
					testutil.CheckResourceAttr("data.neon_organization.test", "plan", "business"),
					testutil.CheckResourceAttr("data.neon_organization.test", "managed_by", "console"),
					testutil.CheckResourceAttr("data.neon_organization.test", "allow_hipaa_projects", "false"),
				),
			},
		},
	})
}
