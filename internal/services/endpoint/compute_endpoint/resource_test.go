package compute_endpoint_test

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const endpointJSON = `{
	"id": "ep-test-001",
	"project_id": "test-project-id",
	"branch_id": "br-test-001",
	"type": "read_write",
	"autoscaling_limit_min_cu": 0.25,
	"autoscaling_limit_max_cu": 1,
	"suspend_timeout_seconds": 300,
	"pooler_enabled": true,
	"pooler_mode": "transaction",
	"disabled": false,
	"passwordless_access": false,
	"host": "ep-test-001.us-east-1.aws.neon.tech",
	"region_id": "aws-us-east-1",
	"current_state": "idle",
	"creation_source": "console",
	"settings": {},
	"provisioner": "k8s-neonvm",
	"proxy_host": "us-east-1.aws.neon.tech",
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z"
}`

const endpointUpdatedJSON = `{
	"id": "ep-test-001",
	"project_id": "test-project-id",
	"branch_id": "br-test-001",
	"type": "read_write",
	"name": "renamed-endpoint",
	"autoscaling_limit_min_cu": 0.25,
	"autoscaling_limit_max_cu": 1,
	"suspend_timeout_seconds": 300,
	"pooler_enabled": true,
	"pooler_mode": "transaction",
	"disabled": false,
	"passwordless_access": false,
	"host": "ep-test-001.us-east-1.aws.neon.tech",
	"region_id": "aws-us-east-1",
	"current_state": "active",
	"creation_source": "console",
	"settings": {},
	"provisioner": "k8s-neonvm",
	"proxy_host": "us-east-1.aws.neon.tech",
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-02T00:00:00Z"
}`

func setupEndpointMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints",
		testutil.JSONResponder(201, `{"endpoint": `+endpointJSON+`, "operations": []}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints/ep-test-001",
		testutil.JSONResponder(200, `{"endpoint": `+endpointJSON+`}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints/ep-test-001",
		testutil.JSONResponder(200, `{"endpoint": `+endpointJSON+`, "operations": []}`),
	)
}

// setupEndpointMocksWithUpdate is like setupEndpointMocks, but once the
// endpoint has been updated via PATCH, subsequent GET requests return the
// post-update representation instead of the original one. This is needed to
// exercise the Update path (and the post-apply refresh plan) realistically.
func setupEndpointMocksWithUpdate(transport *httpmock.MockTransport) {
	currentJSON := endpointJSON

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints",
		testutil.JSONResponder(201, `{"endpoint": `+endpointJSON+`, "operations": []}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints/ep-test-001",
		func(req *http.Request) (*http.Response, error) {
			return testutil.JSONResponder(200, `{"endpoint": `+currentJSON+`}`)(req)
		},
	)

	transport.RegisterResponder(http.MethodPatch,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints/ep-test-001",
		func(req *http.Request) (*http.Response, error) {
			currentJSON = endpointUpdatedJSON
			return testutil.JSONResponder(200, `{"endpoint": `+endpointUpdatedJSON+`, "operations": []}`)(req)
		},
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints/ep-test-001",
		func(req *http.Request) (*http.Response, error) {
			return testutil.JSONResponder(200, `{"endpoint": `+currentJSON+`, "operations": []}`)(req)
		},
	)
}

func TestEndpointResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupEndpointMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_endpoint" "test" {
  project_id             = "test-project-id"
  branch_id              = "br-test-001"
  type                   = "read_write"
  autoscaling_limit_min_cu = 0.25
  autoscaling_limit_max_cu = 1
  suspend_timeout_seconds  = 300
  disabled                 = false
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_endpoint.test", "id", "ep-test-001"),
					testutil.CheckResourceAttr("neon_endpoint.test", "host", "ep-test-001.us-east-1.aws.neon.tech"),
					testutil.CheckResourceAttr("neon_endpoint.test", "type", "read_write"),
					testutil.CheckResourceAttr("neon_endpoint.test", "autoscaling_limit_min_cu", "0.25"),
				),
			},
		},
	})
}

// TestEndpointResource_Update verifies that changing a config attribute
// (name) and re-applying does not fail with "Provider produced
// inconsistent result after apply" even though the API also changes
// volatile computed attributes (updated_at, current_state) as a side
// effect of the update.
func TestEndpointResource_Update(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupEndpointMocksWithUpdate(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_endpoint" "test" {
  project_id             = "test-project-id"
  branch_id              = "br-test-001"
  type                   = "read_write"
  autoscaling_limit_min_cu = 0.25
  autoscaling_limit_max_cu = 1
  suspend_timeout_seconds  = 300
  disabled                 = false
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_endpoint.test", "current_state", "idle"),
					testutil.CheckResourceAttr("neon_endpoint.test", "updated_at", "2025-01-01T00:00:00Z"),
				),
			},
			{
				Config: testutil.TestConfig(`
resource "neon_endpoint" "test" {
  project_id             = "test-project-id"
  branch_id              = "br-test-001"
  type                   = "read_write"
  name                     = "renamed-endpoint"
  autoscaling_limit_min_cu = 0.25
  autoscaling_limit_max_cu = 1
  suspend_timeout_seconds  = 300
  disabled                 = false
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_endpoint.test", "name", "renamed-endpoint"),
					testutil.CheckResourceAttr("neon_endpoint.test", "current_state", "active"),
					testutil.CheckResourceAttr("neon_endpoint.test", "updated_at", "2025-01-02T00:00:00Z"),
				),
			},
		},
	})
}

func TestEndpointResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupEndpointMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_endpoint" "test" {
  project_id             = "test-project-id"
  branch_id              = "br-test-001"
  type                   = "read_write"
  autoscaling_limit_min_cu = 0.25
  autoscaling_limit_max_cu = 1
  suspend_timeout_seconds  = 300
  disabled                 = false
}
`),
			},
			{
				ResourceName:      "neon_endpoint.test",
				ImportState:       true,
				ImportStateId:     "test-project-id/ep-test-001",
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccComputeEndpointResource_Retry423OnCreate verifies that a 423
// Locked response on endpoint create (e.g. because a preceding branch
// create in the same apply left the project's operations still in
// progress) is retried by the provider's HTTP client and the create still
// succeeds, instead of surfacing as an error to Terraform.
func TestAccComputeEndpointResource_Retry423OnCreate(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	createCallCount := 0
	var seenBodies []string
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints",
		func(req *http.Request) (*http.Response, error) {
			createCallCount++

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read request body on attempt %d: %v", createCallCount, err)
			}
			seenBodies = append(seenBodies, string(body))
			// Restore the body so it can still be decoded by the mocked
			// response path below (httpmock doesn't need it, but keep the
			// request well-formed for parity with a real transport).
			req.Body = io.NopCloser(strings.NewReader(string(body)))

			if len(body) == 0 {
				t.Fatalf("attempt %d: request body was empty", createCallCount)
			}
			if !strings.Contains(string(body), `"branch_id":"br-test-001"`) ||
				!strings.Contains(string(body), `"type":"read_write"`) {
				t.Fatalf("attempt %d: request body missing expected fields: %s", createCallCount, body)
			}

			if createCallCount == 1 {
				return testutil.JSONResponder(423, `{"code":"locked","message":"project is locked"}`)(req)
			}
			return testutil.JSONResponder(201, `{"endpoint": `+endpointJSON+`, "operations": []}`)(req)
		},
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints/ep-test-001",
		testutil.JSONResponder(200, `{"endpoint": `+endpointJSON+`}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints/ep-test-001",
		testutil.JSONResponder(200, `{"endpoint": `+endpointJSON+`, "operations": []}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_endpoint" "test" {
  project_id             = "test-project-id"
  branch_id              = "br-test-001"
  type                   = "read_write"
  autoscaling_limit_min_cu = 0.25
  autoscaling_limit_max_cu = 1
  suspend_timeout_seconds  = 300
  disabled                 = false
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_endpoint.test", "id", "ep-test-001"),
					testutil.CheckResourceAttr("neon_endpoint.test", "host", "ep-test-001.us-east-1.aws.neon.tech"),
				),
			},
		},
	})

	if createCallCount != 2 {
		t.Fatalf("got %d create attempts, want 2", createCallCount)
	}
	if len(seenBodies) != 2 {
		t.Fatalf("got %d recorded bodies, want 2", len(seenBodies))
	}
	if seenBodies[0] != seenBodies[1] {
		t.Fatalf("retried POST body differs from original:\nattempt 1: %s\nattempt 2: %s", seenBodies[0], seenBodies[1])
	}
}

// TestEndpointResource_Create_WaitsForOperations verifies that Create polls
// the operations endpoint until the operation returned alongside the
// endpoint reaches a terminal state before returning, so a subsequent
// resource depending on the endpoint (or the practitioner) doesn't observe
// the endpoint before it has actually finished provisioning.
func TestEndpointResource_Create_WaitsForOperations(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	const operationID = "6c4f1d3a-2f2b-4a4a-9d3e-2f6a2f6a2f6a"
	const operationRunningJSON = `{
		"id": "` + operationID + `",
		"project_id": "test-project-id",
		"endpoint_id": "ep-test-001",
		"action": "create_compute",
		"status": "running",
		"failures_count": 0,
		"created_at": "2025-01-01T00:00:00Z",
		"updated_at": "2025-01-01T00:00:00Z",
		"total_duration_ms": 0
	}`
	const operationFinishedJSON = `{
		"id": "` + operationID + `",
		"project_id": "test-project-id",
		"endpoint_id": "ep-test-001",
		"action": "create_compute",
		"status": "finished",
		"failures_count": 0,
		"created_at": "2025-01-01T00:00:00Z",
		"updated_at": "2025-01-01T00:00:01Z",
		"total_duration_ms": 100
	}`

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints",
		testutil.JSONResponder(201, `{"endpoint": `+endpointJSON+`, "operations": [`+operationRunningJSON+`]}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints/ep-test-001",
		testutil.JSONResponder(200, `{"endpoint": `+endpointJSON+`}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints/ep-test-001",
		testutil.JSONResponder(200, `{"endpoint": `+endpointJSON+`, "operations": []}`),
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
resource "neon_endpoint" "test" {
  project_id             = "test-project-id"
  branch_id              = "br-test-001"
  type                   = "read_write"
  autoscaling_limit_min_cu = 0.25
  autoscaling_limit_max_cu = 1
  suspend_timeout_seconds  = 300
  disabled                 = false
}
`),
				Check: testutil.CheckResourceAttr("neon_endpoint.test", "id", "ep-test-001"),
			},
		},
	})

	if operationCallCount < 2 {
		t.Errorf("expected the operations endpoint to be polled at least twice (running, then finished), got %d calls", operationCallCount)
	}
}

// TestEndpointResource_Create_OperationFailureStillPersistsState verifies
// the Create-orphan rule: once the Create API call itself has succeeded,
// the created endpoint's ID and other attributes must be saved to state
// even if the operation that provisions it later fails, so the endpoint
// isn't left orphaned outside Terraform. The apply must still surface the
// operation failure as an error.
//
// Terraform core taints a resource whose Create response carries both a
// non-null new state and error diagnostics, so the next apply destroys and
// recreates it rather than merely refreshing. This test's second POST
// response succeeds (operation "finished") to let that recreate converge,
// and asserts the Delete mock (for the tainted instance) and the second
// Create were both exercised - proving the first Create's result was never
// lost even though it errored.
func TestEndpointResource_Create_OperationFailureStillPersistsState(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	const operationID = "9a1b2c3d-4e5f-4a4a-9d3e-1a2b3c4d5e6f"
	const operationFailedJSON = `{
		"id": "` + operationID + `",
		"project_id": "test-project-id",
		"endpoint_id": "ep-test-001",
		"action": "create_compute",
		"status": "failed",
		"error": "compute provisioning failed",
		"failures_count": 1,
		"created_at": "2025-01-01T00:00:00Z",
		"updated_at": "2025-01-01T00:00:01Z",
		"total_duration_ms": 100
	}`

	createCallCount := 0
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints",
		func(req *http.Request) (*http.Response, error) {
			createCallCount++
			if createCallCount == 1 {
				return testutil.JSONResponder(201, `{"endpoint": `+endpointJSON+`, "operations": [`+operationFailedJSON+`]}`)(req)
			}
			return testutil.JSONResponder(201, `{"endpoint": `+endpointJSON+`, "operations": []}`)(req)
		},
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints/ep-test-001",
		testutil.JSONResponder(200, `{"endpoint": `+endpointJSON+`}`),
	)

	deleteCallCount := 0
	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints/ep-test-001",
		func(req *http.Request) (*http.Response, error) {
			deleteCallCount++
			return testutil.JSONResponder(200, `{"endpoint": `+endpointJSON+`, "operations": []}`)(req)
		},
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_endpoint" "test" {
  project_id             = "test-project-id"
  branch_id              = "br-test-001"
  type                   = "read_write"
  autoscaling_limit_min_cu = 0.25
  autoscaling_limit_max_cu = 1
  suspend_timeout_seconds  = 300
  disabled                 = false
}
`),
				// The operation ends "failed" without ever polling, so this
				// error surfaces on the very first apply. Terraform still
				// taints the resource using the state saved before the
				// error (proving Create did not orphan it).
				ExpectError: regexp.MustCompile(`Endpoint created but operations did not complete`),
			},
			{
				// The tainted resource from step 1 is destroyed (exercising
				// the Delete mock) and recreated; this time the operation
				// finishes successfully.
				Config: testutil.TestConfig(`
resource "neon_endpoint" "test" {
  project_id             = "test-project-id"
  branch_id              = "br-test-001"
  type                   = "read_write"
  autoscaling_limit_min_cu = 0.25
  autoscaling_limit_max_cu = 1
  suspend_timeout_seconds  = 300
  disabled                 = false
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_endpoint.test", "id", "ep-test-001"),
					testutil.CheckResourceAttr("neon_endpoint.test", "host", "ep-test-001.us-east-1.aws.neon.tech"),
				),
			},
		},
	})

	if createCallCount != 2 {
		t.Errorf("expected 2 Create attempts (initial failure + recreate of the tainted resource), got %d", createCallCount)
	}
	// One delete for the tainted-resource replace in step 2, plus one for
	// the framework's final destroy at the end of the test.
	if deleteCallCount < 1 {
		t.Errorf("expected the tainted resource to be destroyed at least once before recreation, got %d delete calls", deleteCallCount)
	}
}

func TestEndpointResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_endpoint" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
  type       = "read_write"
}
`),
				ExpectError: regexp.MustCompile(`Failed to create endpoint`),
			},
		},
	})
}
