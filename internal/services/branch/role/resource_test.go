package role_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const roleJSON = `{
	"branch_id": "br-test-001",
	"name": "myrole",
	"password": "generated-password-123",
	"protected": false,
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z"
}`

func setupRoleMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/roles",
		testutil.JSONResponder(201, `{"role": `+roleJSON+`, "operations": []}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/roles/myrole",
		testutil.JSONResponder(200, `{"role": `+roleJSON+`}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/roles/myrole",
		testutil.JSONResponder(200, `{"role": `+roleJSON+`, "operations": []}`),
	)
}

func TestRoleResource_CreateReadDelete(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupRoleMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_role" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "myrole"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_role.test", "name", "myrole"),
					testutil.CheckResourceAttr("neon_role.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("neon_role.test", "password", "generated-password-123"),
				),
			},
		},
	})
}

func TestRoleResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupRoleMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_role" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "myrole"
}
`),
			},
			{
				ResourceName:                         "neon_role.test",
				ImportState:                          true,
				ImportStateId:                        "test-project-id/br-test-001/myrole",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"password"},
			},
		},
	})
}

func TestRoleResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/roles",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_role" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "myrole"
}
`),
				ExpectError: regexp.MustCompile(`Failed to create role`),
			},
		},
	})
}
