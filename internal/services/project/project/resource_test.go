package project_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const projectJSON = `{
	"id": "test-project-id",
	"name": "my-project",
	"region_id": "aws-us-east-1",
	"pg_version": 16,
	"history_retention_seconds": 86400,
	"store_passwords": true,
	"platform_id": "aws",
	"provisioner": "k8s-neonvm",
	"proxy_host": "us-east-1.aws.neon.tech",
	"branch_logical_size_limit": 0,
	"branch_logical_size_limit_bytes": 0,
	"data_storage_bytes_hour": 0,
	"data_transfer_bytes": 0,
	"written_data_bytes": 0,
	"compute_time_seconds": 0,
	"active_time_seconds": 0,
	"cpu_used_sec": 0,
	"creation_source": "console",
	"owner_id": "owner-001",
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z",
	"consumption_period_start": "2025-01-01T00:00:00Z",
	"consumption_period_end": "2025-02-01T00:00:00Z",
	"settings": {
		"quota": {},
		"allowed_ips": {
			"ips": ["0.0.0.0/0"],
			"protected_branches_only": false
		},
		"enable_logical_replication": false,
		"block_public_connections": false,
		"block_vpc_connections": false
	},
	"default_endpoint_settings": {
		"autoscaling_limit_min_cu": 0.25,
		"autoscaling_limit_max_cu": 0.25,
		"suspend_timeout_seconds": 300
	}
}`

const branchMinJSON = `{"id":"br-main","project_id":"test-project-id","name":"main","current_state":"init","state_changed_at":"2025-01-01T00:00:00Z","creation_source":"console","primary":false,"default":true,"protected":false,"cpu_used_sec":0,"compute_time_seconds":0,"active_time_seconds":0,"written_data_bytes":0,"data_transfer_bytes":0,"created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z"}`

func setupProjectMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects",
		testutil.JSONResponder(201, `{
			"project": `+projectJSON+`,
			"connection_uris": [],
			"roles": [],
			"databases": [],
			"operations": [],
			"branch": `+branchMinJSON+`,
			"endpoints": []
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(200, `{"project": `+projectJSON+`}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(200, `{"project": `+projectJSON+`}`),
	)
}

func TestProjectResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupProjectMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_project.test", "id", "test-project-id"),
					testutil.CheckResourceAttr("neon_project.test", "name", "my-project"),
					testutil.CheckResourceAttr("neon_project.test", "region_id", "aws-us-east-1"),
					testutil.CheckResourceAttr("neon_project.test", "pg_version", "16"),
					testutil.CheckResourceAttr("neon_project.test", "store_passwords", "true"),
					testutil.CheckResourceAttr("neon_project.test", "history_retention_seconds", "86400"),
					testutil.CheckResourceAttr("neon_project.test", "compute_provisioner", "k8s-neonvm"),
					testutil.CheckResourceAttr("neon_project.test", "default_endpoint_settings.autoscaling_limit_min_cu", "0.25"),
					testutil.CheckResourceAttr("neon_project.test", "default_endpoint_settings.autoscaling_limit_max_cu", "0.25"),
					testutil.CheckResourceAttr("neon_project.test", "default_endpoint_settings.suspend_timeout_seconds", "300"),
					testutil.CheckResourceAttr("neon_project.test", "settings.enable_logical_replication", "false"),
					testutil.CheckResourceAttr("neon_project.test", "settings.block_public_connections", "false"),
					testutil.CheckResourceAttr("neon_project.test", "settings.block_vpc_connections", "false"),
					testutil.CheckResourceAttr("neon_project.test", "settings.allowed_ips.ips.#", "1"),
					testutil.CheckResourceAttr("neon_project.test", "settings.allowed_ips.ips.0", "0.0.0.0/0"),
					testutil.CheckResourceAttr("neon_project.test", "settings.allowed_ips.protected_branches_only", "false"),
				),
			},
		},
	})
}

func TestProjectResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupProjectMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project"
}
`),
			},
			{
				ResourceName:      "neon_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestProjectResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects",
		testutil.JSONResponder(403, `{"message":"authentication error","code":"AUTH_FAILED"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project"
}
`),
				ExpectError: regexp.MustCompile(`Failed to create project`),
			},
		},
	})
}
