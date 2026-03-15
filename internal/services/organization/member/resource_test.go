package member_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const memberRoleJSON = `{
	"id": "d5a5a5a5-b4b4-c3c3-d2d2-e1e1e1e1e1e1",
	"user_id": "a1a1a1a1-b2b2-c3c3-d4d4-e5e5e5e5e5e5",
	"org_id": "org-test-001",
	"role": "member",
	"joined_at": "2025-01-01T00:00:00Z"
}`

const memberRoleAdminJSON = `{
	"id": "d5a5a5a5-b4b4-c3c3-d2d2-e1e1e1e1e1e1",
	"user_id": "a1a1a1a1-b2b2-c3c3-d4d4-e5e5e5e5e5e5",
	"org_id": "org-test-001",
	"role": "admin",
	"joined_at": "2025-01-01T00:00:00Z"
}`

const memberURL = "https://neon.example.com/api/v2/organizations/org-test-001/members/d5a5a5a5-b4b4-c3c3-d2d2-e1e1e1e1e1e1"

func setupMemberRoleMocks(transport *httpmock.MockTransport, roleJSON string) {
	transport.RegisterResponder(http.MethodPatch, memberURL,
		testutil.JSONResponder(200, roleJSON),
	)
	transport.RegisterResponder(http.MethodGet, memberURL,
		testutil.JSONResponder(200, roleJSON),
	)
}

func TestOrganizationMemberRoleResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupMemberRoleMocks(transport, memberRoleJSON)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_organization_member_role" "test" {
  org_id = "org-test-001"
  member_id       = "d5a5a5a5-b4b4-c3c3-d2d2-e1e1e1e1e1e1"
  role            = "member"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_organization_member_role.test", "org_id", "org-test-001"),
					testutil.CheckResourceAttr("neon_organization_member_role.test", "member_id", "d5a5a5a5-b4b4-c3c3-d2d2-e1e1e1e1e1e1"),
					testutil.CheckResourceAttr("neon_organization_member_role.test", "role", "member"),
					testutil.CheckResourceAttr("neon_organization_member_role.test", "user_id", "a1a1a1a1-b2b2-c3c3-d4d4-e5e5e5e5e5e5"),
					testutil.CheckResourceAttr("neon_organization_member_role.test", "joined_at", "2025-01-01T00:00:00Z"),
				),
			},
		},
	})
}

func TestOrganizationMemberRoleResource_Update(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupMemberRoleMocks(transport, memberRoleJSON)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_organization_member_role" "test" {
  org_id = "org-test-001"
  member_id       = "d5a5a5a5-b4b4-c3c3-d2d2-e1e1e1e1e1e1"
  role            = "member"
}
`),
				Check: testutil.CheckResourceAttr("neon_organization_member_role.test", "role", "member"),
			},
			{
				PreConfig: func() {
					setupMemberRoleMocks(transport, memberRoleAdminJSON)
				},
				Config: testutil.TestConfig(`
resource "neon_organization_member_role" "test" {
  org_id = "org-test-001"
  member_id       = "d5a5a5a5-b4b4-c3c3-d2d2-e1e1e1e1e1e1"
  role            = "admin"
}
`),
				Check: testutil.CheckResourceAttr("neon_organization_member_role.test", "role", "admin"),
			},
		},
	})
}

func TestOrganizationMemberRoleResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupMemberRoleMocks(transport, memberRoleJSON)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_organization_member_role" "test" {
  org_id = "org-test-001"
  member_id       = "d5a5a5a5-b4b4-c3c3-d2d2-e1e1e1e1e1e1"
  role            = "member"
}
`),
			},
			{
				ResourceName:                         "neon_organization_member_role.test",
				ImportState:                          true,
				ImportStateId:                        "org-test-001/d5a5a5a5-b4b4-c3c3-d2d2-e1e1e1e1e1e1",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "member_id",
			},
		},
	})
}

func TestOrganizationMemberRoleResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPatch, memberURL,
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_organization_member_role" "test" {
  org_id = "org-test-001"
  member_id       = "d5a5a5a5-b4b4-c3c3-d2d2-e1e1e1e1e1e1"
  role            = "member"
}
`),
				ExpectError: regexp.MustCompile(`Failed to update organization member role`),
			},
		},
	})
}
