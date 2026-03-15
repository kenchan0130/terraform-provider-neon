package neon_auth_test

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

const neonAuthCreateResponseJSON = `{
	"auth_provider": "stack_v2",
	"auth_provider_project_id": "auth-proj-001",
	"pub_client_key": "pub-key",
	"secret_server_key": "secret-key",
	"jwks_url": "https://example.com/.well-known/jwks.json",
	"schema_name": "neon_auth",
	"table_name": "users_sync",
	"base_url": "https://auth.example.com"
}`

func setupNeonAuthMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth",
		testutil.JSONResponder(201, neonAuthCreateResponseJSON),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth",
		testutil.JSONResponder(200, neonAuthIntegrationJSON),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth",
		testutil.JSONResponder(200, `{}`),
	)
}

func TestNeonAuthResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupNeonAuthMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  auth_provider = "stack_v2"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_branch_neon_auth.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("neon_branch_neon_auth.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("neon_branch_neon_auth.test", "auth_provider", "stack_v2"),
					testutil.CheckResourceAttr("neon_branch_neon_auth.test", "auth_provider_project_id", "auth-proj-001"),
					testutil.CheckResourceAttr("neon_branch_neon_auth.test", "db_name", "neondb"),
					testutil.CheckResourceAttr("neon_branch_neon_auth.test", "jwks_url", "https://example.com/.well-known/jwks.json"),
					testutil.CheckResourceAttr("neon_branch_neon_auth.test", "base_url", "https://auth.example.com"),
				),
			},
		},
	})
}

func TestNeonAuthResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupNeonAuthMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  auth_provider = "stack_v2"
}
`),
			},
			{
				ResourceName:                         "neon_branch_neon_auth.test",
				ImportState:                          true,
				ImportStateId:                        "test-project-id/br-test-001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "branch_id",
				ImportStateVerifyIgnore:              []string{"database_name"},
			},
		},
	})
}

func TestNeonAuthResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  auth_provider = "stack_v2"
}
`),
				ExpectError: regexp.MustCompile(`Failed to create NeonAuth integration`),
			},
		},
	})
}
