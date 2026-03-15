package branch_schema_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestBranchSchemaDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-branch-id/schema",
		testutil.JSONResponder(200, `{"sql": "CREATE TABLE users (id serial PRIMARY KEY, name text NOT NULL);"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_branch_schema" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-branch-id"
  database_name = "neondb"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_branch_schema.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("data.neon_branch_schema.test", "branch_id", "br-test-branch-id"),
					testutil.CheckResourceAttr("data.neon_branch_schema.test", "database_name", "neondb"),
					testutil.CheckResourceAttr("data.neon_branch_schema.test", "sql", "CREATE TABLE users (id serial PRIMARY KEY, name text NOT NULL);"),
				),
			},
		},
	})
}
