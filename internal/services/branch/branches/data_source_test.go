package branches_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const branchesJSON = `{
	"branches": [
		{
			"id": "br-test-001",
			"project_id": "test-project-id",
			"name": "main",
			"current_state": "ready",
			"state_changed_at": "2025-01-01T00:00:00Z",
			"creation_source": "console",
			"primary": false,
			"default": true,
			"protected": false,
			"cpu_used_sec": 0,
			"compute_time_seconds": 0,
			"active_time_seconds": 0,
			"written_data_bytes": 0,
			"data_transfer_bytes": 0,
			"created_at": "2025-01-01T00:00:00Z",
			"updated_at": "2025-01-02T00:00:00Z"
		},
		{
			"id": "br-test-002",
			"project_id": "test-project-id",
			"name": "dev-branch",
			"parent_id": "br-test-001",
			"current_state": "ready",
			"state_changed_at": "2025-01-01T00:00:00Z",
			"creation_source": "console",
			"primary": false,
			"default": false,
			"protected": false,
			"cpu_used_sec": 0,
			"compute_time_seconds": 0,
			"active_time_seconds": 0,
			"written_data_bytes": 0,
			"data_transfer_bytes": 0,
			"created_at": "2025-01-02T00:00:00Z",
			"updated_at": "2025-01-03T00:00:00Z"
		}
	],
	"annotations": {},
	"pagination": {}
}`

func TestBranchesDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches",
		testutil.JSONResponder(200, branchesJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_branches" "test" {
  project_id = "test-project-id"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_branches.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("data.neon_branches.test", "branches.#", "2"),
					testutil.CheckResourceAttr("data.neon_branches.test", "branches.0.id", "br-test-001"),
					testutil.CheckResourceAttr("data.neon_branches.test", "branches.0.name", "main"),
					testutil.CheckResourceAttr("data.neon_branches.test", "branches.0.current_state", "ready"),
					testutil.CheckResourceAttr("data.neon_branches.test", "branches.0.created_at", "2025-01-01T00:00:00Z"),
					testutil.CheckResourceAttr("data.neon_branches.test", "branches.0.updated_at", "2025-01-02T00:00:00Z"),
					testutil.CheckResourceAttr("data.neon_branches.test", "branches.1.id", "br-test-002"),
					testutil.CheckResourceAttr("data.neon_branches.test", "branches.1.name", "dev-branch"),
					testutil.CheckResourceAttr("data.neon_branches.test", "branches.1.parent_id", "br-test-001"),
					testutil.CheckResourceAttr("data.neon_branches.test", "branches.1.current_state", "ready"),
					testutil.CheckResourceAttr("data.neon_branches.test", "branches.1.created_at", "2025-01-02T00:00:00Z"),
					testutil.CheckResourceAttr("data.neon_branches.test", "branches.1.updated_at", "2025-01-03T00:00:00Z"),
				),
			},
		},
	})
}
