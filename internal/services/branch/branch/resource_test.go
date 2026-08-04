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

// TestBranchResource_Update_WaitsForOperations verifies that Update polls
// the operations endpoint until the operation returned alongside the PATCH
// response reaches a terminal state before returning, and that the final
// state is refreshed from the API afterward (read-back) rather than just
// reusing the PATCH response.
//
// The PATCH response ("transitional") and the post-wait GET response
// ("settled") are deliberately different: transitional keeps current_state
// "init" and pre-wait (zero) usage counters, while settled reports
// current_state "ready" and counters that advanced while the operation was
// in flight (as would happen on a real, active branch). If Update only
// used the PATCH response - or if the volatile int64 counters still used
// UseStateForUnknown instead of planmodifiers.UnknownOnResourceChangeInt64
// - this test would fail: either the Check assertions below would see the
// stale transitional values, or apply itself would fail with "Provider
// produced inconsistent result after apply" because the plan carried the
// prior (zero) counter values forward as "known" while Update tried to
// write the advanced ones.
//
// It also records the ORDER of calls and asserts the settled GET happens
// strictly after the operation poll that reports "finished", proving the
// read-back is real and not just a coincidentally-consistent mock.
func TestBranchResource_Update_WaitsForOperations(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	const operationID = "1c2d3e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f"
	const operationRunningJSON = `{
		"id": "` + operationID + `",
		"project_id": "test-project-id",
		"branch_id": "br-test-001",
		"action": "apply_config",
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
		"action": "apply_config",
		"status": "finished",
		"failures_count": 0,
		"created_at": "2025-01-01T00:00:00Z",
		"updated_at": "2025-01-01T00:00:01Z",
		"total_duration_ms": 100
	}`

	// The PATCH response: operation still has to run, usage counters not
	// yet advanced.
	branchTransitionalJSON := `{
		"id": "br-test-001",
		"project_id": "test-project-id",
		"parent_id": "br-parent-001",
		"parent_lsn": "0/1B482A0",
		"parent_timestamp": "2025-01-01T00:00:00Z",
		"name": "dev-branch",
		"slug": "br-test-001",
		"project_slug": "test-project-id",
		"current_state": "init",
		"state_changed_at": "2025-01-01T00:00:00Z",
		"creation_source": "console",
		"primary": false,
		"default": false,
		"protected": true,
		"cpu_used_sec": 0,
		"compute_time_seconds": 0,
		"active_time_seconds": 0,
		"written_data_bytes": 0,
		"data_transfer_bytes": 0,
		"created_at": "2025-01-01T00:00:00Z",
		"updated_at": "2025-01-01T00:00:00Z",
		"init_source": "parent-data"
	}`

	// The post-wait read-back GET response: operation has finished, and the
	// branch's own usage counters advanced while it was in flight (as they
	// would on an active branch).
	branchSettledJSON := `{
		"id": "br-test-001",
		"project_id": "test-project-id",
		"parent_id": "br-parent-001",
		"parent_lsn": "0/1B482A0",
		"parent_timestamp": "2025-01-01T00:00:00Z",
		"name": "dev-branch",
		"slug": "br-test-001",
		"project_slug": "test-project-id",
		"current_state": "ready",
		"state_changed_at": "2025-01-01T00:00:02Z",
		"creation_source": "console",
		"primary": false,
		"default": false,
		"protected": true,
		"cpu_used_sec": 120,
		"compute_time_seconds": 120,
		"active_time_seconds": 90,
		"written_data_bytes": 4096,
		"data_transfer_bytes": 2048,
		"logical_size": 8192,
		"created_at": "2025-01-01T00:00:00Z",
		"updated_at": "2025-01-01T00:00:02Z",
		"init_source": "parent-data"
	}`

	var events []string

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

	// operationsSettled flips to true only once the operation poll below
	// has reported "finished", so the GET responder can distinguish a
	// legitimate post-wait read-back from any other GET (e.g. the initial
	// create's implicit refresh, or the test framework's pre-apply plan
	// refresh) that must still see the pre-update representation.
	operationsSettled := false
	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001",
		func(req *http.Request) (*http.Response, error) {
			if operationsSettled {
				events = append(events, "GET:settled")
				return testutil.JSONResponder(200, `{"branch": `+branchSettledJSON+`, "annotation": {"object": {"type": "branch", "id": "br-test-001"}, "value": {}}}`)(req)
			}
			events = append(events, "GET:initial")
			return testutil.JSONResponder(200, `{"branch": `+branchJSON+`, "annotation": {"object": {"type": "branch", "id": "br-test-001"}, "value": {}}}`)(req)
		},
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001",
		testutil.JSONResponder(200, `{"branch": `+branchSettledJSON+`, "operations": []}`),
	)

	transport.RegisterResponder(http.MethodPatch,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001",
		func(req *http.Request) (*http.Response, error) {
			events = append(events, "PATCH")
			return testutil.JSONResponder(200, `{"branch": `+branchTransitionalJSON+`, "operations": [`+operationRunningJSON+`]}`)(req)
		},
	)

	operationCallCount := 0
	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/operations/"+operationID,
		func(req *http.Request) (*http.Response, error) {
			operationCallCount++
			if operationCallCount == 1 {
				events = append(events, "OPPOLL:running")
				return testutil.JSONResponder(200, `{"operation": `+operationRunningJSON+`}`)(req)
			}
			operationsSettled = true
			events = append(events, "OPPOLL:finished")
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
  protected  = false
}
`),
			},
			{
				// Toggling protected forces an Update (PATCH) call, whose
				// mocked response carries the running-then-finished
				// operation above.
				Config: testutil.TestConfig(`
resource "neon_branch" "test" {
  project_id = "test-project-id"
  name       = "dev-branch"
  protected  = true
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_branch.test", "id", "br-test-001"),
					testutil.CheckResourceAttr("neon_branch.test", "protected", "true"),
					// These must reflect the settled read-back, not the
					// transitional PATCH response.
					testutil.CheckResourceAttr("neon_branch.test", "current_state", "ready"),
					testutil.CheckResourceAttr("neon_branch.test", "compute_time_seconds", "120"),
					testutil.CheckResourceAttr("neon_branch.test", "active_time_seconds", "90"),
					testutil.CheckResourceAttr("neon_branch.test", "written_data_bytes", "4096"),
					testutil.CheckResourceAttr("neon_branch.test", "data_transfer_bytes", "2048"),
					testutil.CheckResourceAttr("neon_branch.test", "logical_size", "8192"),
				),
			},
		},
	})

	if operationCallCount < 2 {
		t.Errorf("expected the operations endpoint to be polled at least twice (running, then finished), got %d calls", operationCallCount)
	}

	finishedIdx, settledGetIdx := -1, -1
	for i, e := range events {
		if finishedIdx == -1 && e == "OPPOLL:finished" {
			finishedIdx = i
		}
		if settledGetIdx == -1 && finishedIdx != -1 && e == "GET:settled" {
			settledGetIdx = i
		}
	}
	if finishedIdx == -1 {
		t.Fatalf("expected an OPPOLL:finished event, got event order: %v", events)
	}
	if settledGetIdx == -1 {
		t.Fatalf("expected a GET:settled event after OPPOLL:finished (read-back never happened), got event order: %v", events)
	}
	if settledGetIdx <= finishedIdx {
		t.Errorf("expected the settled read-back GET to happen after the terminal operation poll, got event order: %v", events)
	}
}

// TestBranchResource_Update_ReadBackFailurePreservesPreWaitState verifies
// that when the Update API call and the subsequent operations wait both
// succeed, but the read-back GET that refreshes final state fails, the
// resource keeps the pre-wait state already saved from the PATCH response
// (per the CLAUDE.md rule that any failure after a successful mutating
// call must not discard state already saved) instead of losing track of
// the resource entirely.
func TestBranchResource_Update_ReadBackFailurePreservesPreWaitState(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	const operationID = "5f6a7b8c-9d0e-4f1a-b2c3-4d5e6f7a8b9c"
	const operationFinishedJSON = `{
		"id": "` + operationID + `",
		"project_id": "test-project-id",
		"branch_id": "br-test-001",
		"action": "apply_config",
		"status": "finished",
		"failures_count": 0,
		"created_at": "2025-01-01T00:00:00Z",
		"updated_at": "2025-01-01T00:00:01Z",
		"total_duration_ms": 100
	}`

	branchProtectedJSON := `{
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
		"protected": true,
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
			"branch": `+branchJSON+`,
			"endpoints": [],
			"operations": [],
			"roles": [],
			"databases": [],
			"connection_uris": []
		}`),
	)

	// The operations wait succeeds (operation already "finished" in the
	// PATCH response), so Update proceeds to the read-back GET. That
	// read-back GET fails with a 500, simulating a transient failure right
	// after the mutation and its operations already succeeded. The test
	// provider config uses RetryMax: 3, i.e. 4 total attempts per
	// Terraform-level request (see
	// TestBranchDataAPIResource_ReadBackFailureAfterCreate for the same
	// pattern), so all 4 attempts of the read-back's single logical call
	// must fail for the error to actually surface instead of being
	// transparently retried away. Any GET before the PATCH (e.g. Create's
	// own post-wait read-back, or Terraform's automatic pre-plan refresh)
	// must keep succeeding so step 1 and the refresh at the start of step 2
	// aren't affected by this failure injection - only the Update's own
	// read-back is under test.
	patchApplied := false
	readBackFailureAttempts := 0
	readBackFailed := false
	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001",
		func(req *http.Request) (*http.Response, error) {
			if patchApplied && readBackFailureAttempts < 4 {
				readBackFailureAttempts++
				readBackFailed = true
				return testutil.JSONResponder(500, `{"code":"internal_error","message":"internal error"}`)(req)
			}
			body := branchJSON
			if patchApplied {
				body = branchProtectedJSON
			}
			return testutil.JSONResponder(200, `{"branch": `+body+`, "annotation": {"object": {"type": "branch", "id": "br-test-001"}, "value": {}}}`)(req)
		},
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001",
		testutil.JSONResponder(200, `{"branch": `+branchProtectedJSON+`, "operations": []}`),
	)

	transport.RegisterResponder(http.MethodPatch,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001",
		func(req *http.Request) (*http.Response, error) {
			patchApplied = true
			return testutil.JSONResponder(200, `{"branch": `+branchProtectedJSON+`, "operations": [`+operationFinishedJSON+`]}`)(req)
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
  protected  = false
}
`),
			},
			{
				// The read-back GET fails on its first call after the
				// PATCH, so this step's apply errors even though the PATCH
				// and the operations wait both succeeded. Per the
				// Create/Update-orphan rule, the PATCH response
				// (branchProtectedJSON: protected=true) must already be
				// saved to state before this error surfaces.
				Config: testutil.TestConfig(`
resource "neon_branch" "test" {
  project_id = "test-project-id"
  name       = "dev-branch"
  protected  = true
}
`),
				ExpectError: regexp.MustCompile(`failed to read back the final state`),
			},
			{
				// Re-applying identical config triggers Terraform's
				// automatic pre-plan refresh (Read), which now succeeds
				// (readBackFailed is already true) and returns the same
				// values already saved from step 2's PATCH response. Since
				// "protected" already matches config, no further
				// diff/Update occurs; the Check below reads the refreshed
				// state directly and proves it still holds the PATCH
				// response's values rather than having reverted to step 1's
				// state or been lost.
				Config: testutil.TestConfig(`
resource "neon_branch" "test" {
  project_id = "test-project-id"
  name       = "dev-branch"
  protected  = true
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_branch.test", "id", "br-test-001"),
					testutil.CheckResourceAttr("neon_branch.test", "protected", "true"),
				),
			},
		},
	})

	if !readBackFailed {
		t.Error("expected the read-back GET to have been attempted and to have failed once")
	}
}

// TestBranchResource_Delete_WaitsForOperations verifies that Delete polls
// the operations endpoint until the operation returned alongside the DELETE
// response reaches a terminal state before returning.
func TestBranchResource_Delete_WaitsForOperations(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	const operationID = "2b3c4d5e-6f7a-4b8c-9d0e-1f2a3b4c5d6e"
	const operationRunningJSON = `{
		"id": "` + operationID + `",
		"project_id": "test-project-id",
		"branch_id": "br-test-001",
		"action": "delete_timeline",
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
		"action": "delete_timeline",
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
		testutil.JSONResponder(200, `{"branch": `+branchJSON+`, "operations": [`+operationRunningJSON+`]}`),
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
			},
		},
	})

	if operationCallCount < 2 {
		t.Errorf("expected the operations endpoint to be polled at least twice (running, then finished) during destroy, got %d calls", operationCallCount)
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
