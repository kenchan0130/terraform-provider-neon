package jwks_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const jwksJSON = `{
	"id": "jwks-test-001",
	"project_id": "test-project-id",
	"jwks_url": "https://example.com/.well-known/jwks.json",
	"provider_name": "Clerk",
	"branch_id": "br-test-001",
	"jwt_audience": "neon",
	"role_names": [],
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z"
}`

func setupJWKSMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/jwks",
		testutil.JSONResponder(201, `{
			"jwks": `+jwksJSON+`,
			"operations": []
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/jwks",
		testutil.JSONResponder(200, `{"jwks": [`+jwksJSON+`]}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/jwks/jwks-test-001",
		testutil.JSONResponder(200, jwksJSON),
	)
}

func TestJWKSResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupJWKSMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project_jwks" "test" {
  project_id    = "test-project-id"
  jwks_url      = "https://example.com/.well-known/jwks.json"
  provider_name = "Clerk"
  branch_id     = "br-test-001"
  jwt_audience  = "neon"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_project_jwks.test", "id", "jwks-test-001"),
					testutil.CheckResourceAttr("neon_project_jwks.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("neon_project_jwks.test", "jwks_url", "https://example.com/.well-known/jwks.json"),
					testutil.CheckResourceAttr("neon_project_jwks.test", "provider_name", "Clerk"),
					testutil.CheckResourceAttr("neon_project_jwks.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("neon_project_jwks.test", "jwt_audience", "neon"),
				),
			},
		},
	})
}

func TestJWKSResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupJWKSMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project_jwks" "test" {
  project_id    = "test-project-id"
  jwks_url      = "https://example.com/.well-known/jwks.json"
  provider_name = "Clerk"
  branch_id     = "br-test-001"
  jwt_audience  = "neon"
}
`),
			},
			{
				ResourceName:      "neon_project_jwks.test",
				ImportState:       true,
				ImportStateId:     "test-project-id/jwks-test-001",
				ImportStateVerify: true,
			},
		},
	})
}

func TestJWKSResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/jwks",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project_jwks" "test" {
  project_id    = "test-project-id"
  jwks_url      = "https://example.com/.well-known/jwks.json"
  provider_name = "Clerk"
}
`),
				ExpectError: regexp.MustCompile(`Failed to create JWKS`),
			},
		},
	})
}
