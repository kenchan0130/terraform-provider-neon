package compute_endpoints_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const endpointsJSON = `{
	"endpoints": [
		{
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
		},
		{
			"id": "ep-test-002",
			"project_id": "test-project-id",
			"branch_id": "br-test-002",
			"type": "read_only",
			"name": "my-read-replica",
			"autoscaling_limit_min_cu": 0.5,
			"autoscaling_limit_max_cu": 2,
			"suspend_timeout_seconds": 600,
			"pooler_enabled": false,
			"pooler_mode": "transaction",
			"disabled": false,
			"passwordless_access": true,
			"host": "ep-test-002.us-east-1.aws.neon.tech",
			"region_id": "aws-us-east-1",
			"current_state": "active",
			"creation_source": "console",
			"settings": {},
			"provisioner": "k8s-neonvm",
			"proxy_host": "us-east-1.aws.neon.tech",
			"created_at": "2025-02-01T00:00:00Z",
			"updated_at": "2025-02-15T00:00:00Z"
		}
	]
}`

func TestEndpointsDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/endpoints",
		testutil.JSONResponder(200, endpointsJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_endpoints" "test" {
  project_id = "test-project-id"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.#", "2"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.0.id", "ep-test-001"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.0.branch_id", "br-test-001"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.0.type", "read_write"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.0.host", "ep-test-001.us-east-1.aws.neon.tech"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.0.current_state", "idle"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.0.autoscaling_limit_min_cu", "0.25"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.0.autoscaling_limit_max_cu", "1"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.0.pooler_enabled", "true"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.0.disabled", "false"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.1.id", "ep-test-002"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.1.branch_id", "br-test-002"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.1.type", "read_only"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.1.name", "my-read-replica"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.1.host", "ep-test-002.us-east-1.aws.neon.tech"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.1.current_state", "active"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.1.autoscaling_limit_min_cu", "0.5"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.1.autoscaling_limit_max_cu", "2"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.1.pooler_enabled", "false"),
					testutil.CheckResourceAttr("data.neon_endpoints.test", "endpoints.1.passwordless_access", "true"),
				),
			},
		},
	})
}
