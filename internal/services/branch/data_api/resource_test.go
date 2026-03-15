package data_api_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const dataAPIGetResponseJSON = `{
	"url": "https://data-api.example.com/test-db",
	"status": "active",
	"settings": null,
	"available_schemas": null
}`

func setupDataAPIMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/data-api/test-db",
		testutil.JSONResponder(201, `{"url": "https://data-api.example.com/test-db"}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/data-api/test-db",
		testutil.JSONResponder(200, dataAPIGetResponseJSON),
	)

	transport.RegisterResponder(http.MethodPut,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/data-api/test-db",
		testutil.JSONResponder(201, `{}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/data-api/test-db",
		testutil.JSONResponder(200, `{}`),
	)
}

func TestBranchDataAPIResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupDataAPIMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_data_api" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  database_name = "test-db"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_branch_data_api.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("neon_branch_data_api.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("neon_branch_data_api.test", "database_name", "test-db"),
					testutil.CheckResourceAttr("neon_branch_data_api.test", "url", "https://data-api.example.com/test-db"),
					testutil.CheckResourceAttr("neon_branch_data_api.test", "status", "active"),
				),
			},
		},
	})
}

func TestBranchDataAPIResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupDataAPIMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_data_api" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  database_name = "test-db"
}
`),
			},
			{
				ResourceName:                         "neon_branch_data_api.test",
				ImportState:                          true,
				ImportStateId:                        "test-project-id/br-test-001/test-db",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "project_id",
			},
		},
	})
}

func TestBranchDataAPIResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/data-api/test-db",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch_data_api" "test" {
  project_id    = "test-project-id"
  branch_id     = "br-test-001"
  database_name = "test-db"
}
`),
				ExpectError: regexp.MustCompile(`Failed to create branch data API`),
			},
		},
	})
}
