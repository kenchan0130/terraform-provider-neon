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
	"authentication_method": "password",
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
					testutil.CheckResourceAttr("neon_role.test", "protected", "false"),
					testutil.CheckResourceAttr("neon_role.test", "authentication_method", "password"),
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
			},
		},
	})
}

const noLoginRoleJSON = `{
	"branch_id": "br-test-001",
	"name": "myrole",
	"protected": false,
	"authentication_method": "no_login",
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z"
}`

func setupNoLoginRoleMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/roles",
		testutil.JSONResponder(201, `{"role": `+noLoginRoleJSON+`, "operations": []}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/roles/myrole",
		testutil.JSONResponder(200, `{"role": `+noLoginRoleJSON+`}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/roles/myrole",
		testutil.JSONResponder(200, `{"role": `+noLoginRoleJSON+`, "operations": []}`),
	)
}

// TestRoleResource_ImportNoLogin verifies that importing a NOLOGIN role and
// declaring no_login = true in config results in an empty plan (no forced
// replacement), because Read derives no_login from authentication_method.
func TestRoleResource_ImportNoLogin(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupNoLoginRoleMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_role" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "myrole"
  no_login   = true
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_role.test", "no_login", "true"),
					testutil.CheckResourceAttr("neon_role.test", "authentication_method", "no_login"),
				),
			},
			{
				ResourceName:                         "neon_role.test",
				ImportState:                          true,
				ImportStateId:                        "test-project-id/br-test-001/myrole",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				Config: testutil.TestConfig(`
resource "neon_role" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "myrole"
  no_login   = true
}
`),
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
