package projects_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const projectsJSON = `{
	"projects": [
		{
			"id": "project-001",
			"platform_id": "aws",
			"region_id": "aws-us-east-2",
			"name": "my-project",
			"provisioner": "k8s-neonvm",
			"pg_version": 16,
			"proxy_host": "us-east-2.aws.neon.tech",
			"branch_logical_size_limit": 0,
			"branch_logical_size_limit_bytes": 0,
			"store_passwords": true,
			"active_time": 100,
			"cpu_used_sec": 0,
			"creation_source": "console",
			"created_at": "2025-01-01T00:00:00Z",
			"updated_at": "2025-06-15T12:00:00Z",
			"owner_id": "user-001"
		},
		{
			"id": "project-002",
			"platform_id": "aws",
			"region_id": "aws-eu-central-1",
			"name": "another-project",
			"provisioner": "k8s-neonvm",
			"pg_version": 15,
			"proxy_host": "eu-central-1.aws.neon.tech",
			"branch_logical_size_limit": 0,
			"branch_logical_size_limit_bytes": 0,
			"store_passwords": true,
			"active_time": 200,
			"cpu_used_sec": 0,
			"creation_source": "console",
			"created_at": "2025-03-10T08:30:00Z",
			"updated_at": "2025-07-20T16:45:00Z",
			"owner_id": "user-001"
		}
	],
	"unavailable_project_ids": [],
	"applications": {},
	"integrations": {}
}`

func TestProjectsDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects",
		testutil.JSONResponder(200, projectsJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_projects" "test" {
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_projects.test", "projects.#", "2"),
					testutil.CheckResourceAttr("data.neon_projects.test", "projects.0.id", "project-001"),
					testutil.CheckResourceAttr("data.neon_projects.test", "projects.0.name", "my-project"),
					testutil.CheckResourceAttr("data.neon_projects.test", "projects.0.region_id", "aws-us-east-2"),
					testutil.CheckResourceAttr("data.neon_projects.test", "projects.0.pg_version", "16"),
					testutil.CheckResourceAttr("data.neon_projects.test", "projects.0.created_at", "2025-01-01T00:00:00Z"),
					testutil.CheckResourceAttr("data.neon_projects.test", "projects.0.updated_at", "2025-06-15T12:00:00Z"),
					testutil.CheckResourceAttr("data.neon_projects.test", "projects.1.id", "project-002"),
					testutil.CheckResourceAttr("data.neon_projects.test", "projects.1.name", "another-project"),
					testutil.CheckResourceAttr("data.neon_projects.test", "projects.1.region_id", "aws-eu-central-1"),
					testutil.CheckResourceAttr("data.neon_projects.test", "projects.1.pg_version", "15"),
					testutil.CheckResourceAttr("data.neon_projects.test", "projects.1.created_at", "2025-03-10T08:30:00Z"),
					testutil.CheckResourceAttr("data.neon_projects.test", "projects.1.updated_at", "2025-07-20T16:45:00Z"),
				),
			},
		},
	})
}
