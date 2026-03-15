package vpc_endpoint_restriction_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func setupVPCEndpointRestrictionMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/vpc_endpoints/vpce-test-001",
		testutil.JSONResponder(200, `{}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/vpc_endpoints",
		testutil.JSONResponder(200, `{"endpoints": [{"vpc_endpoint_id": "vpce-test-001", "label": "my-vpc-endpoint"}]}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/vpc_endpoints/vpce-test-001",
		testutil.JSONResponder(200, `{}`),
	)
}

func TestVPCEndpointRestrictionResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupVPCEndpointRestrictionMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project_vpc_endpoint_restriction" "test" {
  project_id      = "test-project-id"
  vpc_endpoint_id = "vpce-test-001"
  label           = "my-vpc-endpoint"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_project_vpc_endpoint_restriction.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("neon_project_vpc_endpoint_restriction.test", "vpc_endpoint_id", "vpce-test-001"),
					testutil.CheckResourceAttr("neon_project_vpc_endpoint_restriction.test", "label", "my-vpc-endpoint"),
				),
			},
		},
	})
}

func TestVPCEndpointRestrictionResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupVPCEndpointRestrictionMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project_vpc_endpoint_restriction" "test" {
  project_id      = "test-project-id"
  vpc_endpoint_id = "vpce-test-001"
  label           = "my-vpc-endpoint"
}
`),
			},
			{
				ResourceName:                         "neon_project_vpc_endpoint_restriction.test",
				ImportState:                          true,
				ImportStateId:                        "test-project-id/vpce-test-001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "vpc_endpoint_id",
			},
		},
	})
}

func TestVPCEndpointRestrictionResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/vpc_endpoints/vpce-test-001",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project_vpc_endpoint_restriction" "test" {
  project_id      = "test-project-id"
  vpc_endpoint_id = "vpce-test-001"
  label           = "my-vpc-endpoint"
}
`),
				ExpectError: regexp.MustCompile(`Failed to assign VPC endpoint restriction`),
			},
		},
	})
}
