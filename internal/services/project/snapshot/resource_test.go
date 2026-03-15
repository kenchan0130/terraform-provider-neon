package snapshot_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
