package database_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const databaseJSON = `{
	"id": 12345,
	"branch_id": "br-test-001",
	"name": "mydb",
	"owner_name": "myrole",
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z"
}`

func setupDatabaseMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/databases",
		testutil.JSONResponder(201, `{"database": `+databaseJSON+`, "operations": []}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/databases/mydb",
		testutil.JSONResponder(200, `{"database": `+databaseJSON+`}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/databases/mydb",
		testutil.JSONResponder(200, `{"database": `+databaseJSON+`, "operations": []}`),
	)
}

func TestDatabaseResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupDatabaseMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_database" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "mydb"
  owner_name = "myrole"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_database.test", "id", "12345"),
					testutil.CheckResourceAttr("neon_database.test", "name", "mydb"),
					testutil.CheckResourceAttr("neon_database.test", "owner_name", "myrole"),
				),
			},
		},
	})
}

func TestDatabaseResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupDatabaseMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_database" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "mydb"
  owner_name = "myrole"
}
`),
			},
			{
				ResourceName:      "neon_database.test",
				ImportState:       true,
				ImportStateId:     "test-project-id/br-test-001/mydb",
				ImportStateVerify: true,
			},
		},
	})
}

func TestDatabaseResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/databases",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_database" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "mydb"
  owner_name = "myrole"
}
`),
				ExpectError: regexp.MustCompile(`Failed to create database`),
			},
		},
	})
}
