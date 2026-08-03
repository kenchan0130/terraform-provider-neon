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
