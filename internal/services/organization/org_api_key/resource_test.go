package org_api_key_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func setupOrgApiKeyMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/organizations/org-test-id/api_keys",
		testutil.JSONResponder(200, `{
			"id": 67890,
			"key": "neon-org-api-key-secret-value",
			"name": "my-org-api-key",
			"created_at": "2025-01-01T00:00:00Z",
			"created_by": "00000000-0000-0000-0000-000000000000"
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/organizations/org-test-id/api_keys",
		testutil.JSONResponder(200, `[{
			"id": 67890,
			"name": "my-org-api-key",
			"created_at": "2025-01-01T00:00:00Z",
			"created_by": {"id": "00000000-0000-0000-0000-000000000000", "name": "test-user", "image": ""},
			"last_used_at": null,
			"last_used_from_addr": ""
		}]`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/organizations/org-test-id/api_keys/67890",
		testutil.JSONResponder(200, `{
			"id": 67890,
			"name": "my-org-api-key",
			"created_at": "2025-01-01T00:00:00Z",
			"created_by": "00000000-0000-0000-0000-000000000000",
			"last_used_from_addr": "",
			"revoked": true
		}`),
	)
}

func TestOrgApiKeyResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupOrgApiKeyMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_organization_api_key" "test" {
  org_id = "org-test-id"
  name   = "my-org-api-key"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_organization_api_key.test", "id", "67890"),
					testutil.CheckResourceAttr("neon_organization_api_key.test", "org_id", "org-test-id"),
					testutil.CheckResourceAttr("neon_organization_api_key.test", "name", "my-org-api-key"),
					testutil.CheckResourceAttr("neon_organization_api_key.test", "key", "neon-org-api-key-secret-value"),
				),
			},
		},
	})
}

func TestOrgApiKeyResource_WithProjectID(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/organizations/org-test-id/api_keys",
		testutil.JSONResponder(200, `{
			"id": 67891,
			"key": "neon-org-api-key-project-scoped",
			"name": "my-project-api-key",
			"created_at": "2025-01-01T00:00:00Z",
			"created_by": "00000000-0000-0000-0000-000000000000",
			"project_id": "test-project-id"
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/organizations/org-test-id/api_keys",
		testutil.JSONResponder(200, `[{
			"id": 67891,
			"name": "my-project-api-key",
			"created_at": "2025-01-01T00:00:00Z",
			"created_by": {"id": "00000000-0000-0000-0000-000000000000", "name": "test-user", "image": ""},
			"last_used_at": null,
			"last_used_from_addr": "",
			"project_id": "test-project-id"
		}]`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/organizations/org-test-id/api_keys/67891",
		testutil.JSONResponder(200, `{
			"id": 67891,
			"name": "my-project-api-key",
			"created_at": "2025-01-01T00:00:00Z",
			"created_by": "00000000-0000-0000-0000-000000000000",
			"last_used_from_addr": "",
			"revoked": true
		}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_organization_api_key" "test" {
  org_id     = "org-test-id"
  name       = "my-project-api-key"
  project_id = "test-project-id"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_organization_api_key.test", "id", "67891"),
					testutil.CheckResourceAttr("neon_organization_api_key.test", "name", "my-project-api-key"),
					testutil.CheckResourceAttr("neon_organization_api_key.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("neon_organization_api_key.test", "key", "neon-org-api-key-project-scoped"),
				),
			},
		},
	})
}

func TestOrgApiKeyResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/organizations/org-test-id/api_keys",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_organization_api_key" "test" {
  org_id = "org-test-id"
  name   = "my-org-api-key"
}
`),
				ExpectError: regexp.MustCompile(`Failed to create organization API key`),
			},
		},
	})
}
