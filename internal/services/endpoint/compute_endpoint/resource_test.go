package compute_endpoint_test

import (
	"net/http"
	"regexp"
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
