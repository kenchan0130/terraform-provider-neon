package vpc_endpoint_restriction_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestVPCEndpointRestrictionDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/vpc_endpoints",
		testutil.JSONResponder(200, `{"endpoints": [{"vpc_endpoint_id": "vpce-test-001", "label": "my-vpc-endpoint"}]}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_project_vpc_endpoint_restriction" "test" {
  project_id      = "test-project-id"
  vpc_endpoint_id = "vpce-test-001"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_project_vpc_endpoint_restriction.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("data.neon_project_vpc_endpoint_restriction.test", "vpc_endpoint_id", "vpce-test-001"),
					testutil.CheckResourceAttr("data.neon_project_vpc_endpoint_restriction.test", "label", "my-vpc-endpoint"),
				),
			},
		},
	})
}
