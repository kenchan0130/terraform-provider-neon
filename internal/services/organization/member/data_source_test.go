package member_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const organizationMembersJSON = `{
	"members": [
		{
			"member": {
				"id": "a1a1a1a1-b2b2-c3c3-d4d4-e5e5e5e5e5e5",
				"user_id": "11111111-2222-3333-4444-555555555555",
				"org_id": "org-test-001",
				"role": "admin",
				"joined_at": "2025-01-01T00:00:00Z"
			},
			"user": {
				"email": "admin@example.com",
				"has_mfa": true
			}
		},
		{
			"member": {
				"id": "f1f1f1f1-a2a2-b3b3-c4c4-d5d5d5d5d5d5",
				"user_id": "66666666-7777-8888-9999-aaaaaaaaaaaa",
				"org_id": "org-test-001",
				"role": "member",
				"joined_at": "2025-06-15T12:30:00Z"
			},
			"user": {
				"email": "member@example.com",
				"has_mfa": false
			}
		}
	],
	"pagination": {}
}`

func TestOrganizationMembersDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/organizations/org-test-001/members",
		testutil.JSONResponder(200, organizationMembersJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_organization_members" "test" {
  org_id = "org-test-001"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_organization_members.test", "org_id", "org-test-001"),
					testutil.CheckResourceAttr("data.neon_organization_members.test", "members.#", "2"),
					testutil.CheckResourceAttr("data.neon_organization_members.test", "members.0.id", "a1a1a1a1-b2b2-c3c3-d4d4-e5e5e5e5e5e5"),
					testutil.CheckResourceAttr("data.neon_organization_members.test", "members.0.user_id", "11111111-2222-3333-4444-555555555555"),
					testutil.CheckResourceAttr("data.neon_organization_members.test", "members.0.role", "admin"),
					testutil.CheckResourceAttr("data.neon_organization_members.test", "members.0.email", "admin@example.com"),
					testutil.CheckResourceAttr("data.neon_organization_members.test", "members.0.has_mfa", "true"),
					testutil.CheckResourceAttr("data.neon_organization_members.test", "members.0.joined_at", "2025-01-01T00:00:00Z"),
					testutil.CheckResourceAttr("data.neon_organization_members.test", "members.1.id", "f1f1f1f1-a2a2-b3b3-c4c4-d5d5d5d5d5d5"),
					testutil.CheckResourceAttr("data.neon_organization_members.test", "members.1.role", "member"),
					testutil.CheckResourceAttr("data.neon_organization_members.test", "members.1.email", "member@example.com"),
					testutil.CheckResourceAttr("data.neon_organization_members.test", "members.1.has_mfa", "false"),
				),
			},
		},
	})
}

func TestOrganizationMembersDataSource_ReadWithQuery(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/organizations/org-test-001/members",
		testutil.JSONResponder(200, organizationMembersJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_organization_members" "test" {
  org_id = "org-test-001"

  query = {
    sort_by    = "email"
    sort_order = "asc"
  }
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_organization_members.test", "members.#", "2"),
				),
			},
		},
	})
}
