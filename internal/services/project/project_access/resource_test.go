package project_access_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestProjectAccessResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/proj-001/permissions",
		testutil.JSONResponder(200, `{
			"id": "perm-001",
			"granted_to_email": "user@example.com",
			"granted_at": "2025-01-01T00:00:00Z"
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/proj-001/permissions",
		testutil.JSONResponder(200, `{
			"project_permissions": [{
				"id": "perm-001",
				"granted_to_email": "user@example.com",
				"granted_at": "2025-01-01T00:00:00Z"
			}]
		}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/proj-001/permissions/perm-001",
		testutil.JSONResponder(200, `{
			"id": "perm-001",
			"granted_to_email": "user@example.com",
			"granted_at": "2025-01-01T00:00:00Z",
			"revoked_at": "2025-01-02T00:00:00Z"
		}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project_access" "test" {
  project_id       = "proj-001"
  granted_to_email = "user@example.com"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_project_access.test", "project_id", "proj-001"),
					testutil.CheckResourceAttr("neon_project_access.test", "permission_id", "perm-001"),
					testutil.CheckResourceAttr("neon_project_access.test", "granted_to_email", "user@example.com"),
				),
			},
		},
	})
}

func TestProjectAccessResource_ReadRemoved(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	callCount := 0
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/proj-001/permissions",
		testutil.JSONResponder(200, `{
			"id": "perm-001",
			"granted_to_email": "user@example.com",
			"granted_at": "2025-01-01T00:00:00Z"
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/proj-001/permissions",
		func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount <= 1 {
				return testutil.JSONResponder(200, `{
					"project_permissions": [{
						"id": "perm-001",
						"granted_to_email": "user@example.com",
						"granted_at": "2025-01-01T00:00:00Z"
					}]
				}`)(req)
			}
			// Permission revoked externally
			return testutil.JSONResponder(200, `{
				"project_permissions": [{
					"id": "perm-001",
					"granted_to_email": "user@example.com",
					"granted_at": "2025-01-01T00:00:00Z",
					"revoked_at": "2025-01-02T00:00:00Z"
				}]
			}`)(req)
		},
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/proj-001/permissions/perm-001",
		testutil.JSONResponder(200, `{
			"id": "perm-001",
			"granted_to_email": "user@example.com",
			"granted_at": "2025-01-01T00:00:00Z",
			"revoked_at": "2025-01-02T00:00:00Z"
		}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project_access" "test" {
  project_id       = "proj-001"
  granted_to_email = "user@example.com"
}
`),
			},
			{
				Config: testutil.TestConfig(`
resource "neon_project_access" "test" {
  project_id       = "proj-001"
  granted_to_email = "user@example.com"
}
`),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestProjectAccessResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/proj-001/permissions",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project_access" "test" {
  project_id       = "proj-001"
  granted_to_email = "user@example.com"
}
`),
				ExpectError: regexp.MustCompile(`Failed to grant project permission`),
			},
		},
	})
}

func TestProjectAccessResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/proj-001/permissions",
		testutil.JSONResponder(200, `{
			"id": "perm-001",
			"granted_to_email": "user@example.com",
			"granted_at": "2025-01-01T00:00:00Z"
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/proj-001/permissions",
		testutil.JSONResponder(200, `{
			"project_permissions": [{
				"id": "perm-001",
				"granted_to_email": "user@example.com",
				"granted_at": "2025-01-01T00:00:00Z"
			}]
		}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/proj-001/permissions/perm-001",
		testutil.JSONResponder(200, `{
			"id": "perm-001",
			"granted_to_email": "user@example.com",
			"granted_at": "2025-01-01T00:00:00Z",
			"revoked_at": "2025-01-02T00:00:00Z"
		}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project_access" "test" {
  project_id       = "proj-001"
  granted_to_email = "user@example.com"
}
`),
			},
			{
				ResourceName:                         "neon_project_access.test",
				ImportState:                          true,
				ImportStateId:                        "proj-001/perm-001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "permission_id",
			},
		},
	})
}
