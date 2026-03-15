package vpc_endpoint_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestVPCEndpointDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/organizations/org-test-001/vpc/region/aws-us-east-1/vpc_endpoints/vpce-abc123",
		testutil.JSONResponder(200, vpcEndpointDetailsJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_organization_vpc_endpoint" "test" {
  org_id          = "org-test-001"
  region_id       = "aws-us-east-1"
  vpc_endpoint_id = "vpce-abc123"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_organization_vpc_endpoint.test", "vpc_endpoint_id", "vpce-abc123"),
					testutil.CheckResourceAttr("data.neon_organization_vpc_endpoint.test", "label", "my-vpc-endpoint"),
					testutil.CheckResourceAttr("data.neon_organization_vpc_endpoint.test", "state", "new"),
					testutil.CheckResourceAttr("data.neon_organization_vpc_endpoint.test", "num_restricted_projects", "0"),
				),
			},
		},
	})
}
