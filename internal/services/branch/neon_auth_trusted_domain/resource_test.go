package neon_auth_trusted_domain_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const neonAuthIntegrationJSON = `{
	"auth_provider": "stack_v2",
	"auth_provider_project_id": "auth-proj-001",
	"branch_id": "br-test-001",
	"db_name": "neondb",
	"created_at": "2025-01-01T00:00:00Z",
	"owned_by": "neon",
	"jwks_url": "https://example.com/.well-known/jwks.json",
	"base_url": "https://auth.example.com"
}`

func setupTrustedDomainMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth",
		testutil.JSONResponder(200, neonAuthIntegrationJSON),
	)

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/domains",
		testutil.JSONResponder(201, `{}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/domains",
		testutil.JSONResponder(200, `{"domains": [{"domain": "https://example.com", "auth_provider": "stack_v2"}]}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/domains",
		testutil.JSONResponder(200, `{}`),
	)
}

func TestNeonAuthTrustedDomainResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupTrustedDomainMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth_trusted_domain" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  domain     = "https://example.com"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_branch_neon_auth_trusted_domain.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("neon_branch_neon_auth_trusted_domain.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("neon_branch_neon_auth_trusted_domain.test", "domain", "https://example.com"),
					testutil.CheckResourceAttr("neon_branch_neon_auth_trusted_domain.test", "auth_provider", "stack_v2"),
				),
			},
		},
	})
}

func TestNeonAuthTrustedDomainResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupTrustedDomainMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth_trusted_domain" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  domain     = "https://example.com"
}
`),
			},
			{
				ResourceName:                         "neon_branch_neon_auth_trusted_domain.test",
				ImportState:                          true,
				ImportStateId:                        "test-project-id/br-test-001/https://example.com",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "domain",
			},
		},
	})
}

func TestNeonAuthTrustedDomainResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth",
		testutil.JSONResponder(200, neonAuthIntegrationJSON),
	)

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/domains",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth_trusted_domain" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  domain     = "https://example.com"
}
`),
				ExpectError: regexp.MustCompile(`Failed to add NeonAuth trusted domain`),
			},
		},
	})
}
