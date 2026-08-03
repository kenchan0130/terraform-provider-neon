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

// TestBranchResource_ExpiresAtRemovedFromConfigClearsExpiration verifies
// that removing expires_at from the configuration produces a diff and
// triggers an Update call that unsets the expiration (regression test for
// expires_at previously being Optional+Computed, which prevented any diff
// from being generated when the attribute was removed from config).
func TestBranchResource_ExpiresAtRemovedFromConfigClearsExpiration(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupBranchMocks(transport)

	branchWithExpiresJSON := `{
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
		"init_source": "parent-data",
		"expires_at": "2026-12-31T00:00:00Z"
	}`

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches",
		testutil.JSONResponder(201, `{
			"branch": `+branchWithExpiresJSON+`,
			"endpoints": [],
			"operations": [],
			"roles": [],
			"databases": [],
			"connection_uris": []
		}`),
	)

	// Simulate server-side state: expiration starts set, and gets cleared
	// once the expected PATCH (Update) request is made.
	expiresCleared := false

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001",
		func(req *http.Request) (*http.Response, error) {
			body := branchWithExpiresJSON
			if expiresCleared {
				body = branchJSON
			}
			return testutil.JSONResponder(200, `{"branch": `+body+`, "annotation": {"object": {"type": "branch", "id": "br-test-001"}, "value": {}}}`)(req)
		},
	)

	transport.RegisterResponder(http.MethodPatch,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001",
		func(req *http.Request) (*http.Response, error) {
			expiresCleared = true
			return testutil.JSONResponder(200, `{"branch": `+branchJSON+`, "operations": []}`)(req)
		},
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_branch" "test" {
  project_id = "test-project-id"
  name       = "dev-branch"
  expires_at = "2026-12-31T00:00:00Z"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_branch.test", "expires_at", "2026-12-31T00:00:00Z"),
				),
			},
			{
				// Removing expires_at from config must produce a diff and
				// call Update (which sends an explicit null), clearing the
				// expiration. The PATCH mock above only returns branchJSON
				// (no expires_at), so if Update were not called the check
				// below would still see the stale value from step 1's state
				// rather than a cleared one.
				Config: testutil.TestConfig(`
resource "neon_branch" "test" {
  project_id = "test-project-id"
  name       = "dev-branch"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("neon_branch.test", "expires_at"),
				),
			},
		},
	})
}

// TestBranchResource_ParentTimestampNonCanonicalConfigPreserved verifies that
// a practitioner-supplied parent_timestamp using a non-canonical RFC 3339
// representation (here, a value that the API echoes back with the same
// instant) is preserved as configured rather than being overwritten with a
// re-formatted value, which previously caused "Provider produced
// inconsistent result after apply" for non-UTC/non-canonical inputs.
func TestBranchResource_ParentTimestampNonCanonicalConfigPreserved(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	// The API responds with the UTC-normalized equivalent of the configured
	// offset timestamp below (both refer to the same instant).
	branchWithOffsetParentTimestampJSON := `{
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

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches",
		testutil.JSONResponder(201, `{
			"branch": `+branchWithOffsetParentTimestampJSON+`,
			"endpoints": [],
			"operations": [],
			"roles": [],
			"databases": [],
			"connection_uris": []
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001",
		testutil.JSONResponder(200, `{"branch": `+branchWithOffsetParentTimestampJSON+`, "annotation": {"object": {"type": "branch", "id": "br-test-001"}, "value": {}}}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001",
		testutil.JSONResponder(200, `{"branch": `+branchWithOffsetParentTimestampJSON+`, "operations": []}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				// This is the same instant as "2025-01-01T00:00:00Z" but
				// expressed with a non-UTC offset, matching what the mocked
				// API response above resolves to.
				Config: testutil.TestConfig(`
resource "neon_branch" "test" {
  project_id       = "test-project-id"
  name             = "dev-branch"
  parent_timestamp = "2025-01-01T09:00:00+09:00"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_branch.test", "parent_timestamp", "2025-01-01T09:00:00+09:00"),
				),
			},
		},
	})
}

// TestBranchResource_Create_WaitsForOperations verifies that Create polls
// the operations endpoint until the operation returned alongside the branch
// reaches a terminal state before returning, so that dependent resources
// (e.g. an endpoint on the same branch, in the same apply) don't race the
// branch's own provisioning and hit 423 Locked.
func TestBranchResource_Create_WaitsForOperations(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	const operationID = "04fd6a4e-7f36-4606-a0ba-43dbddccc543"
	const operationRunningJSON = `{
		"id": "` + operationID + `",
		"project_id": "test-project-id",
		"branch_id": "br-test-001",
		"action": "create_branch",
		"status": "running",
		"failures_count": 0,
		"created_at": "2025-01-01T00:00:00Z",
		"updated_at": "2025-01-01T00:00:00Z",
		"total_duration_ms": 0
	}`
	const operationFinishedJSON = `{
		"id": "` + operationID + `",
		"project_id": "test-project-id",
		"branch_id": "br-test-001",
		"action": "create_branch",
		"status": "finished",
		"failures_count": 0,
		"created_at": "2025-01-01T00:00:00Z",
		"updated_at": "2025-01-01T00:00:01Z",
		"total_duration_ms": 100
	}`

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branches",
		testutil.JSONResponder(201, `{
			"branch": `+branchJSON+`,
			"endpoints": [],
			"operations": [`+operationRunningJSON+`],
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

	operationCallCount := 0
	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/operations/"+operationID,
		func(req *http.Request) (*http.Response, error) {
			operationCallCount++
			if operationCallCount == 1 {
				return testutil.JSONResponder(200, `{"operation": `+operationRunningJSON+`}`)(req)
			}
			return testutil.JSONResponder(200, `{"operation": `+operationFinishedJSON+`}`)(req)
		},
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
				Check: testutil.CheckResourceAttr("neon_branch.test", "id", "br-test-001"),
			},
		},
	})

	if operationCallCount < 2 {
		t.Errorf("expected the operations endpoint to be polled at least twice (running, then finished), got %d calls", operationCallCount)
	}
}

// TestBranchResource_DeleteAfter404 verifies the CLAUDE.md "Delete + 404
// must be success" rule: if the branch is already gone by the time
// Terraform issues the DELETE (e.g. an async delete operation from a
// previous, retried apply already completed, or it was removed outside
// Terraform), the provider must treat the 404 as a successful delete
// instead of surfacing an error that would make destroy fail forever.
func TestBranchResource_DeleteAfter404(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

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

	deleteCallCount := 0
	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001",
		func(req *http.Request) (*http.Response, error) {
			deleteCallCount++
			return testutil.JSONResponder(404, `{"code":"not_found","message":"branch not found"}`)(req)
		},
	)

	// If Delete did not treat 404 as success, the automatic destroy that
	// resource.UnitTest performs at the end of the test would fail the
	// test with a fatal apply error.
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
		},
	})

	if deleteCallCount == 0 {
		t.Error("expected DeleteProjectBranch to be called during test cleanup")
	}
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
