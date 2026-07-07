package snapshot_test

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const snapshotJSON = `{
	"id": "snap-test-001",
	"name": "my-snapshot",
	"lsn": "0/1234567",
	"source_branch_id": "br-test-001",
	"created_at": "2025-01-01T00:00:00Z",
	"expires_at": "2025-12-31T23:59:59Z",
	"manual": true
}`

func setupSnapshotMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/snapshot",
		testutil.JSONResponder(200, `{
			"snapshot": `+snapshotJSON+`,
			"operations": []
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/snapshots",
		testutil.JSONResponder(200, `{"snapshots": [`+snapshotJSON+`]}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/snapshots/snap-test-001",
		testutil.JSONResponder(202, `{"operations": []}`),
	)
}

func TestSnapshotResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupSnapshotMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_snapshot" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "my-snapshot"
  expires_at = "2025-12-31T23:59:59Z"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_snapshot.test", "id", "snap-test-001"),
					testutil.CheckResourceAttr("neon_snapshot.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("neon_snapshot.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("neon_snapshot.test", "name", "my-snapshot"),
					testutil.CheckResourceAttr("neon_snapshot.test", "source_branch_id", "br-test-001"),
					testutil.CheckResourceAttr("neon_snapshot.test", "manual", "true"),
				),
			},
		},
	})
}

func TestSnapshotResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupSnapshotMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_snapshot" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "my-snapshot"
  expires_at = "2025-12-31T23:59:59Z"
}
`),
			},
			{
				ResourceName:            "neon_snapshot.test",
				ImportState:             true,
				ImportStateId:           "test-project-id/snap-test-001",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"branch_id"},
			},
		},
	})
}

func TestSnapshotResource_UpdateExpiresAt(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupSnapshotMocks(transport)

	updatedSnapshotJSON := `{
		"id": "snap-test-001",
		"name": "my-snapshot",
		"lsn": "0/1234567",
		"source_branch_id": "br-test-001",
		"created_at": "2025-01-01T00:00:00Z",
		"expires_at": "2026-01-31T23:59:59Z",
		"manual": true
	}`

	updated := false

	transport.RegisterResponder(http.MethodPatch,
		"https://neon.example.com/api/v2/projects/test-project-id/snapshots/snap-test-001",
		func(req *http.Request) (*http.Response, error) {
			updated = true
			return testutil.JSONResponder(200, `{"snapshot": `+updatedSnapshotJSON+`}`)(req)
		},
	)

	// After the update, refresh reads must observe the updated expires_at.
	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/snapshots",
		func(req *http.Request) (*http.Response, error) {
			if updated {
				return testutil.JSONResponder(200, `{"snapshots": [`+updatedSnapshotJSON+`]}`)(req)
			}
			return testutil.JSONResponder(200, `{"snapshots": [`+snapshotJSON+`]}`)(req)
		},
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_snapshot" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "my-snapshot"
  expires_at = "2025-12-31T23:59:59Z"
}
`),
				Check: testutil.CheckResourceAttr("neon_snapshot.test", "expires_at", "2025-12-31T23:59:59Z"),
			},
			{
				// Changing expires_at must be handled as an in-place update (PATCH),
				// not a destroy+recreate, since it does not carry RequiresReplace.
				Config: testutil.TestConfig(`
resource "neon_snapshot" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "my-snapshot"
  expires_at = "2026-01-31T23:59:59Z"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_snapshot.test", "id", "snap-test-001"),
					testutil.CheckResourceAttr("neon_snapshot.test", "expires_at", "2026-01-31T23:59:59Z"),
				),
			},
		},
	})
}

func TestSnapshotResource_ClearExpiresAt(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupSnapshotMocks(transport)

	clearedSnapshotJSON := `{
		"id": "snap-test-001",
		"name": "my-snapshot",
		"lsn": "0/1234567",
		"source_branch_id": "br-test-001",
		"created_at": "2025-01-01T00:00:00Z",
		"manual": true
	}`

	var patchBody string
	updated := false

	transport.RegisterResponder(http.MethodPatch,
		"https://neon.example.com/api/v2/projects/test-project-id/snapshots/snap-test-001",
		func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			patchBody = string(body)
			updated = true
			return testutil.JSONResponder(200, `{"snapshot": `+clearedSnapshotJSON+`}`)(req)
		},
	)

	// After the update, refresh reads must observe expires_at cleared.
	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/snapshots",
		func(req *http.Request) (*http.Response, error) {
			if updated {
				return testutil.JSONResponder(200, `{"snapshots": [`+clearedSnapshotJSON+`]}`)(req)
			}
			return testutil.JSONResponder(200, `{"snapshots": [`+snapshotJSON+`]}`)(req)
		},
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_snapshot" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "my-snapshot"
  expires_at = "2025-12-31T23:59:59Z"
}
`),
				Check: testutil.CheckResourceAttr("neon_snapshot.test", "expires_at", "2025-12-31T23:59:59Z"),
			},
			{
				// Removing expires_at from config must clear it via the API
				// (sent as JSON null), not silently carry forward the prior
				// state value, and must not trigger an inconsistent-result
				// error after apply.
				Config: testutil.TestConfig(`
resource "neon_snapshot" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "my-snapshot"
}
`),
				Check: func(_ *terraform.State) error {
					if !strings.Contains(patchBody, `"expires_at":null`) {
						return fmt.Errorf("expected PATCH body to contain \"expires_at\":null, got: %s", patchBody)
					}
					return nil
				},
			},
		},
	})
}

func TestSnapshotResource_ReadParentProjectDeleted(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/snapshot",
		testutil.JSONResponder(200, `{
			"snapshot": `+snapshotJSON+`,
			"operations": []
		}`),
	)

	callCount := 0
	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/snapshots",
		func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount <= 1 {
				return testutil.JSONResponder(200, `{"snapshots": [`+snapshotJSON+`]}`)(req)
			}
			// The parent project was deleted outside of Terraform.
			return testutil.JSONResponder(404, `{
				"code": "not_found",
				"message": "project not found"
			}`)(req)
		},
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/snapshots/snap-test-001",
		testutil.JSONResponder(202, `{"operations": []}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_snapshot" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "my-snapshot"
  expires_at = "2025-12-31T23:59:59Z"
}
`),
			},
			{
				Config: testutil.TestConfig(`
resource "neon_snapshot" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "my-snapshot"
  expires_at = "2025-12-31T23:59:59Z"
}
`),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestSnapshotResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/snapshot",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_snapshot" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  name       = "my-snapshot"
}
`),
				ExpectError: regexp.MustCompile(`Failed to create snapshot`),
			},
		},
	})
}
