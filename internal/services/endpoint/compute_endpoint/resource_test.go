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
  pooler_enabled           = true
  pooler_mode              = "transaction"
  disabled                 = false
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_endpoint.test", "id", "ep-test-001"),
					testutil.CheckResourceAttr("neon_endpoint.test", "host", "ep-test-001.us-east-1.aws.neon.tech"),
					testutil.CheckResourceAttr("neon_endpoint.test", "type", "read_write"),
					testutil.CheckResourceAttr("neon_endpoint.test", "pooler_enabled", "true"),
					testutil.CheckResourceAttr("neon_endpoint.test", "autoscaling_limit_min_cu", "0.25"),
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
  pooler_enabled           = true
  pooler_mode              = "transaction"
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
