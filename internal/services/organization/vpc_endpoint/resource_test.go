package vpc_endpoint_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const vpcEndpointDetailsJSON = `{
	"vpc_endpoint_id": "vpce-abc123",
	"label": "my-vpc-endpoint",
	"state": "new",
	"num_restricted_projects": 0,
	"example_restricted_projects": []
}`

func setupVPCEndpointMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/organizations/org-test-001/vpc/region/aws-us-east-1/vpc_endpoints/vpce-abc123",
		testutil.JSONResponder(200, `{}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/organizations/org-test-001/vpc/region/aws-us-east-1/vpc_endpoints/vpce-abc123",
		testutil.JSONResponder(200, vpcEndpointDetailsJSON),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/organizations/org-test-001/vpc/region/aws-us-east-1/vpc_endpoints/vpce-abc123",
		testutil.JSONResponder(200, `{}`),
	)
}

func TestVPCEndpointResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupVPCEndpointMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_organization_vpc_endpoint" "test" {
  org_id          = "org-test-001"
  region_id       = "aws-us-east-1"
  vpc_endpoint_id = "vpce-abc123"
  label           = "my-vpc-endpoint"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_organization_vpc_endpoint.test", "vpc_endpoint_id", "vpce-abc123"),
					testutil.CheckResourceAttr("neon_organization_vpc_endpoint.test", "label", "my-vpc-endpoint"),
					testutil.CheckResourceAttr("neon_organization_vpc_endpoint.test", "state", "new"),
					testutil.CheckResourceAttr("neon_organization_vpc_endpoint.test", "num_restricted_projects", "0"),
				),
			},
		},
	})
}

func TestVPCEndpointResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupVPCEndpointMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_organization_vpc_endpoint" "test" {
  org_id          = "org-test-001"
  region_id       = "aws-us-east-1"
  vpc_endpoint_id = "vpce-abc123"
  label           = "my-vpc-endpoint"
}
`),
			},
			{
				ResourceName:                         "neon_organization_vpc_endpoint.test",
				ImportState:                          true,
				ImportStateId:                        "org-test-001/aws-us-east-1/vpce-abc123",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "vpc_endpoint_id",
			},
		},
	})
}

func TestVPCEndpointResource_UpdateServerSideValuesChange(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/organizations/org-test-001/vpc/region/aws-us-east-1/vpc_endpoints/vpce-abc123",
		testutil.JSONResponder(200, `{}`),
	)

	getCount := 0
	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/organizations/org-test-001/vpc/region/aws-us-east-1/vpc_endpoints/vpce-abc123",
		func(req *http.Request) (*http.Response, error) {
			getCount++
			// The first few reads (create + pre-update refreshes) return the
			// original values. Once the update is applied (which calls
			// AssignOrganizationVPCEndpoint followed by a read), the API
			// starts returning the server-side changed values (state
			// transitioned to "accepted", a restriction was added, etc.).
			if getCount <= 3 {
				return testutil.JSONResponder(200, vpcEndpointDetailsJSON)(req)
			}
			return testutil.JSONResponder(200, `{
				"vpc_endpoint_id": "vpce-abc123",
				"label": "my-vpc-endpoint-updated",
				"state": "accepted",
				"num_restricted_projects": 1,
				"example_restricted_projects": ["test-project-id"]
			}`)(req)
		},
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/organizations/org-test-001/vpc/region/aws-us-east-1/vpc_endpoints/vpce-abc123",
		testutil.JSONResponder(200, `{}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_organization_vpc_endpoint" "test" {
  org_id          = "org-test-001"
  region_id       = "aws-us-east-1"
  vpc_endpoint_id = "vpce-abc123"
  label           = "my-vpc-endpoint"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_organization_vpc_endpoint.test", "state", "new"),
					testutil.CheckResourceAttr("neon_organization_vpc_endpoint.test", "num_restricted_projects", "0"),
				),
			},
			{
				// Even though the server-side state/num_restricted_projects/
				// example_restricted_projects change as a side effect of the
				// label-only update, this must not produce a "Provider
				// produced inconsistent result after apply" error.
				Config: testutil.TestConfig(`
resource "neon_organization_vpc_endpoint" "test" {
  org_id          = "org-test-001"
  region_id       = "aws-us-east-1"
  vpc_endpoint_id = "vpce-abc123"
  label           = "my-vpc-endpoint-updated"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_organization_vpc_endpoint.test", "label", "my-vpc-endpoint-updated"),
					testutil.CheckResourceAttr("neon_organization_vpc_endpoint.test", "state", "accepted"),
					testutil.CheckResourceAttr("neon_organization_vpc_endpoint.test", "num_restricted_projects", "1"),
				),
			},
		},
	})
}

func TestVPCEndpointResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/organizations/org-test-001/vpc/region/aws-us-east-1/vpc_endpoints/vpce-abc123",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_organization_vpc_endpoint" "test" {
  org_id          = "org-test-001"
  region_id       = "aws-us-east-1"
  vpc_endpoint_id = "vpce-abc123"
  label           = "my-vpc-endpoint"
}
`),
				ExpectError: regexp.MustCompile(`Failed to assign VPC endpoint`),
			},
		},
	})
}
