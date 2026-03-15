package project_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestProjectDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(200, `{"project": `+projectJSON+`}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_project" "test" {
  id = "test-project-id"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_project.test", "id", "test-project-id"),
					testutil.CheckResourceAttr("data.neon_project.test", "name", "my-project"),
					testutil.CheckResourceAttr("data.neon_project.test", "region_id", "aws-us-east-1"),
					testutil.CheckResourceAttr("data.neon_project.test", "pg_version", "16"),
					testutil.CheckResourceAttr("data.neon_project.test", "compute_provisioner", "k8s-neonvm"),
					testutil.CheckResourceAttr("data.neon_project.test", "default_endpoint_settings.autoscaling_limit_min_cu", "0.25"),
					testutil.CheckResourceAttr("data.neon_project.test", "settings.enable_logical_replication", "false"),
				),
			},
		},
	})
}
