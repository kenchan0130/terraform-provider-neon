package project_access_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestProjectAccessDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/proj-001/permissions",
		testutil.JSONResponder(200, `{
			"project_permissions": [{
				"id": "perm-001",
				"granted_to_email": "user@example.com",
				"granted_at": "2025-01-01T00:00:00Z"
			}]
		}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_project_access" "test" {
  project_id    = "proj-001"
  permission_id = "perm-001"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_project_access.test", "project_id", "proj-001"),
					testutil.CheckResourceAttr("data.neon_project_access.test", "permission_id", "perm-001"),
					testutil.CheckResourceAttr("data.neon_project_access.test", "granted_to_email", "user@example.com"),
				),
			},
		},
	})
}
