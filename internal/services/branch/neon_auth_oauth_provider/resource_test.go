package neon_auth_oauth_provider_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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
		func(req *http.Request) (*http.Response, error) {
			var body struct {
				ID           string `json:"id"`
				ClientSecret string `json:"client_secret"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				resp := httpmock.NewStringResponse(400, `{"message":"invalid request"}`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			}
			if body.ID == "github" {
				resp := httpmock.NewStringResponse(200, `{
					"id": "github",
					"type": "standard",
					"client_id": "my-client-id",
					"client_secret": "my-client-secret"
				}`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			}
			if body.ID != "google" {
				resp := httpmock.NewStringResponse(400, `{"code":"invalid_request","message":"id: must be one of google, github, microsoft, vercel."}`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			}
			if body.ClientSecret == "" {
				resp := httpmock.NewStringResponse(400, `{"code":"invalid_request","message":"client_secret: cannot be blank."}`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			}
			resp := httpmock.NewStringResponse(200, oauthProviderJSON)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		},
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
  id            = "google"
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
  id                       = "google"
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
  id            = "google"
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
  id            = "google"
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
  id                       = "google"
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

func TestNeonAuthOauthProviderResource_ReadNotFoundRemovesResource(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/oauth_providers",
		testutil.JSONResponder(200, oauthProviderJSON),
	)

	// The first GET (the automatic post-apply consistency refresh in step 1)
	// succeeds; subsequent GETs (the explicit refresh in step 2) return 404,
	// simulating the branch/integration having been deleted out-of-band.
	getCalls := 0
	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/oauth_providers",
		func(req *http.Request) (*http.Response, error) {
			getCalls++
			if getCalls == 1 {
				resp := httpmock.NewStringResponse(200, `{"providers": [`+oauthProviderJSON+`]}`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			}
			resp := httpmock.NewStringResponse(404, `{"code":"not_found","message":"branch not found"}`)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		},
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth_oauth_provider" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  id            = "google"
  client_id     = "my-client-id"
  client_secret = "my-client-secret"
}
`),
			},
			{
				// After a 404 on refresh, the resource is removed from state,
				// so a subsequent plan proposes re-creating it.
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestNeonAuthOauthProviderResource_IdChangeRequiresReplace(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupOauthProviderMocks(transport)

	githubProviderJSON := `{
		"id": "github",
		"type": "standard",
		"client_id": "my-client-id",
		"client_secret": "my-client-secret"
	}`

	// Both providers may be listed depending on which one currently exists.
	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/oauth_providers",
		testutil.JSONResponder(200, `{"providers": [`+oauthProviderJSON+`, `+githubProviderJSON+`]}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/auth/oauth_providers/github",
		testutil.JSONResponder(200, `{}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth_oauth_provider" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  id            = "google"
  client_id     = "my-client-id"
  client_secret = "my-client-secret"
}
`),
			},
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth_oauth_provider" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  id            = "github"
  client_id     = "my-client-id"
  client_secret = "my-client-secret"
}
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("neon_branch_neon_auth_oauth_provider.test", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}

func TestNeonAuthOauthProviderResource_InvalidId(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_neon_auth_oauth_provider" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  id            = "not-a-real-provider"
  client_id     = "my-client-id"
  client_secret = "my-client-secret"
}
`),
				ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value Matches|value must be one of`),
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
  id               = "google"
  client_id        = "my-client-id"
  client_secret_wo = "my-client-secret"
}
`),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}
