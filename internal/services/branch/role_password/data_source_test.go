package role_password_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestRolePasswordDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/proj-001/branches/br-001/roles/myrole/reveal_password",
		testutil.JSONResponder(200, `{"password":"secret123"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_role_password" "test" {
  project_id = "proj-001"
  branch_id  = "br-001"
  role_name  = "myrole"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_role_password.test", "password", "secret123"),
				),
			},
		},
	})
}

func TestRolePasswordDataSource_NotFound(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/proj-001/branches/br-001/roles/myrole/reveal_password",
		testutil.JSONResponder(404, `{"code":"ROLE_NOT_FOUND","message":"role not found"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_role_password" "test" {
  project_id = "proj-001"
  branch_id  = "br-001"
  role_name  = "myrole"
}
`),
				ExpectError: regexp.MustCompile(`Role not found`),
			},
		},
	})
}

func TestRolePasswordDataSource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/proj-001/branches/br-001/roles/myrole/reveal_password",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_role_password" "test" {
  project_id = "proj-001"
  branch_id  = "br-001"
  role_name  = "myrole"
}
`),
				ExpectError: regexp.MustCompile(`Failed to get role password`),
			},
		},
	})
}
