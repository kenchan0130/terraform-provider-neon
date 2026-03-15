package neon_auth_oauth_provider_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const oauthProviderJSON = `{
	"id": "google",
	"type": "standard",
	"client_id": "my-client-id",
	"client_secret": "my-client-secret"
}`

func setupOauthProviderMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/oauth_providers",
		testutil.JSONResponder(200, oauthProviderJSON),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/oauth_providers",
		testutil.JSONResponder(200, `{"providers": [`+oauthProviderJSON+`]}`),
	)

	transport.RegisterResponder(http.MethodPatch,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/oauth_providers/google",
		testutil.JSONResponder(200, oauthProviderJSON),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/oauth_providers/google",
		testutil.JSONResponder(200, `{}`),
	)
}

func TestNeonAuthOauthProviderResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupOauthProviderMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth_oauth_provider" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  type          = "standard"
  client_id     = "my-client-id"
  client_secret = "my-client-secret"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_branch_neon_auth_oauth_provider.test", "id", "google"),
					testutil.CheckResourceAttr("neon_branch_neon_auth_oauth_provider.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("neon_branch_neon_auth_oauth_provider.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("neon_branch_neon_auth_oauth_provider.test", "type", "standard"),
					testutil.CheckResourceAttr("neon_branch_neon_auth_oauth_provider.test", "client_secret", "my-client-secret"),
				),
			},
		},
	})
}

func TestNeonAuthOauthProviderResource_CreateWithWriteOnly(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupOauthProviderMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth_oauth_provider" "test" {
  project_id               = "test-project-id"
  branch_id                = "br-test-001"
  type                     = "standard"
  client_id                = "my-client-id"
  client_secret_wo         = "my-client-secret"
  client_secret_wo_version = "1"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_branch_neon_auth_oauth_provider.test", "id", "google"),
					testutil.CheckResourceAttr("neon_branch_neon_auth_oauth_provider.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("neon_branch_neon_auth_oauth_provider.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("neon_branch_neon_auth_oauth_provider.test", "type", "standard"),
					testutil.CheckResourceAttr("neon_branch_neon_auth_oauth_provider.test", "client_secret_wo_version", "1"),
					// client_secret_wo is write-only and not stored in state
				),
			},
		},
	})
}

func TestNeonAuthOauthProviderResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupOauthProviderMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth_oauth_provider" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  type          = "standard"
  client_id     = "my-client-id"
  client_secret = "my-client-secret"
}
`),
			},
			{
				ResourceName:            "neon_branch_neon_auth_oauth_provider.test",
				ImportState:             true,
				ImportStateId:           "test-project-id/br-test-001/google",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
		},
	})
}

func TestNeonAuthOauthProviderResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/oauth_providers",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth_oauth_provider" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  type          = "standard"
  client_id     = "my-client-id"
  client_secret = "my-client-secret"
}
`),
				ExpectError: regexp.MustCompile(`Failed to add NeonAuth OAuth provider`),
			},
		},
	})
}

func TestNeonAuthOauthProviderResource_ConflictingSecrets(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth_oauth_provider" "test" {
  project_id               = "test-project-id"
  branch_id                = "br-test-001"
  type                     = "standard"
  client_id                = "my-client-id"
  client_secret            = "my-client-secret"
  client_secret_wo         = "my-client-secret"
  client_secret_wo_version = "1"
}
`),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

func TestNeonAuthOauthProviderResource_WriteOnlyWithoutVersion(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth_oauth_provider" "test" {
  project_id       = "test-project-id"
  branch_id        = "br-test-001"
  type             = "standard"
  client_id        = "my-client-id"
  client_secret_wo = "my-client-secret"
}
`),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}
