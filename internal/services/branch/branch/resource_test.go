package branch_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const branchJSON = `{
	"id": "br-test-001",
	"project_id": "test-project-id",
	"parent_id": "br-parent-001",
	"parent_lsn": "0/1B482A0",
	"parent_timestamp": "2025-01-01T00:00:00Z",
	"name": "dev-branch",
	"slug": "br-test-001",
	"project_slug": "test-project-id",
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
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z",
	"init_source": "parent-data"
}`

func setupBranchMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches",
		testutil.JSONResponder(201, `{
			"branch": `+branchJSON+`,
			"endpoints": [],
			"operations": [],
			"roles": [],
			"databases": [],
			"connection_uris": []
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001",
		testutil.JSONResponder(200, `{"branch": `+branchJSON+`, "annotation": {"object": {"type": "branch", "id": "br-test-001"}, "value": {}}}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001",
		testutil.JSONResponder(200, `{"branch": `+branchJSON+`, "operations": []}`),
	)
}

func TestBranchResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupBranchMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch" "test" {
  project_id = "test-project-id"
  name       = "dev-branch"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_branch.test", "id", "br-test-001"),
					testutil.CheckResourceAttr("neon_branch.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("neon_branch.test", "name", "dev-branch"),
					testutil.CheckResourceAttr("neon_branch.test", "parent_id", "br-parent-001"),
					testutil.CheckResourceAttr("neon_branch.test", "parent_lsn", "0/1B482A0"),
					testutil.CheckResourceAttr("neon_branch.test", "parent_timestamp", "2025-01-01T00:00:00Z"),
					testutil.CheckResourceAttr("neon_branch.test", "init_source", "parent-data"),
				),
			},
		},
	})
}

func TestBranchResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupBranchMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch" "test" {
  project_id = "test-project-id"
  name       = "dev-branch"
}
`),
			},
			{
				ResourceName:      "neon_branch.test",
				ImportState:       true,
				ImportStateId:     "test-project-id/br-test-001",
				ImportStateVerify: true,
			},
		},
	})
}

func TestBranchResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch" "test" {
  project_id = "test-project-id"
  name       = "dev-branch"
}
`),
				ExpectError: regexp.MustCompile(`Failed to create branch`),
			},
		},
	})
}
