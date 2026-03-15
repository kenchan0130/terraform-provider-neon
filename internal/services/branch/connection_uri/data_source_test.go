package connection_uri_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestConnectionURIDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/connection_uri",
		testutil.JSONResponder(200, `{"uri": "postgres://user:pass@host/dbname"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_connection_uri" "test" {
  project_id    = "test-project-id"
  database_name = "dbname"
  role_name     = "user"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_connection_uri.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("data.neon_connection_uri.test", "database_name", "dbname"),
					testutil.CheckResourceAttr("data.neon_connection_uri.test", "role_name", "user"),
					testutil.CheckResourceAttr("data.neon_connection_uri.test", "uri", "postgres://user:pass@host/dbname"),
				),
			},
		},
	})
}
