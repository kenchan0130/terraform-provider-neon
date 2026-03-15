package invitations_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const organizationInvitationsJSON = `{
	"invitations": [
		{
			"id": "a1a1a1a1-b2b2-c3c3-d4d4-e5e5e5e5e5e5",
			"email": "user1@example.com",
			"org_id": "org-test-001",
			"invited_by": "11111111-2222-3333-4444-555555555555",
			"invited_at": "2025-01-15T10:00:00Z",
			"role": "member"
		},
		{
			"id": "f1f1f1f1-a2a2-b3b3-c4c4-d5d5d5d5d5d5",
			"email": "user2@example.com",
			"org_id": "org-test-001",
			"invited_by": "11111111-2222-3333-4444-555555555555",
			"invited_at": "2025-02-20T14:30:00Z",
			"role": "admin"
		}
	]
}`

func TestOrganizationInvitationsDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/organizations/org-test-001/invitations",
		testutil.JSONResponder(200, organizationInvitationsJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_organization_invitations" "test" {
  org_id = "org-test-001"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_organization_invitations.test", "org_id", "org-test-001"),
					testutil.CheckResourceAttr("data.neon_organization_invitations.test", "invitations.#", "2"),
					testutil.CheckResourceAttr("data.neon_organization_invitations.test", "invitations.0.id", "a1a1a1a1-b2b2-c3c3-d4d4-e5e5e5e5e5e5"),
					testutil.CheckResourceAttr("data.neon_organization_invitations.test", "invitations.0.email", "user1@example.com"),
					testutil.CheckResourceAttr("data.neon_organization_invitations.test", "invitations.0.org_id", "org-test-001"),
					testutil.CheckResourceAttr("data.neon_organization_invitations.test", "invitations.0.invited_by", "11111111-2222-3333-4444-555555555555"),
					testutil.CheckResourceAttr("data.neon_organization_invitations.test", "invitations.0.invited_at", "2025-01-15T10:00:00Z"),
					testutil.CheckResourceAttr("data.neon_organization_invitations.test", "invitations.0.role", "member"),
					testutil.CheckResourceAttr("data.neon_organization_invitations.test", "invitations.1.id", "f1f1f1f1-a2a2-b3b3-c4c4-d5d5d5d5d5d5"),
					testutil.CheckResourceAttr("data.neon_organization_invitations.test", "invitations.1.email", "user2@example.com"),
					testutil.CheckResourceAttr("data.neon_organization_invitations.test", "invitations.1.role", "admin"),
				),
			},
		},
	})
}
