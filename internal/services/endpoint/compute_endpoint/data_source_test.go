package compute_endpoint_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestEndpointDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints/ep-test-001",
		testutil.JSONResponder(200, `{"endpoint": `+endpointJSON+`}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_endpoint" "test" {
  project_id = "test-project-id"
  id         = "ep-test-001"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_endpoint.test", "id", "ep-test-001"),
					testutil.CheckResourceAttr("data.neon_endpoint.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("data.neon_endpoint.test", "host", "ep-test-001.us-east-1.aws.neon.tech"),
				),
			},
		},
	})
}
