package api_key_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func setupApiKeyMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/api_keys",
		testutil.JSONResponder(200, `{
			"id": 12345,
			"key": "neon-api-key-secret-value",
			"name": "my-api-key",
			"created_at": "2025-01-01T00:00:00Z",
			"created_by": "00000000-0000-0000-0000-000000000000"
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/api_keys",
		testutil.JSONResponder(200, `[{
			"id": 12345,
			"name": "my-api-key",
			"created_at": "2025-01-01T00:00:00Z",
			"created_by": {"id": "00000000-0000-0000-0000-000000000000", "name": "test-user", "image": ""},
			"last_used_at": null,
			"last_used_from_addr": ""
		}]`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/api_keys/12345",
		testutil.JSONResponder(200, `{
			"id": 12345,
			"name": "my-api-key",
			"created_at": "2025-01-01T00:00:00Z",
			"created_by": "00000000-0000-0000-0000-000000000000",
			"last_used_from_addr": "",
			"revoked": true
		}`),
	)
}

func TestApiKeyResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupApiKeyMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_api_key" "test" {
  name = "my-api-key"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_api_key.test", "id", "12345"),
					testutil.CheckResourceAttr("neon_api_key.test", "name", "my-api-key"),
					testutil.CheckResourceAttr("neon_api_key.test", "key", "neon-api-key-secret-value"),
				),
			},
		},
	})
}

func TestApiKeyResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/api_keys",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_api_key" "test" {
  name = "my-api-key"
}
`),
				ExpectError: regexp.MustCompile(`Failed to create API key`),
			},
		},
	})
}
