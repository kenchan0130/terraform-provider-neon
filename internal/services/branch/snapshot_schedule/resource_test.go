package snapshot_schedule_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const scheduleGetResponseJSON = `{
	"schedule": [
		{
			"frequency": "daily",
			"hour": 3,
			"retention_seconds": 86400
		}
	]
}`

func setupSnapshotScheduleMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPut,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/backup_schedule",
		testutil.JSONResponder(200, `{}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/backup_schedule",
		testutil.JSONResponder(200, scheduleGetResponseJSON),
	)
}

func TestSnapshotScheduleResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupSnapshotScheduleMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_snapshot_schedule" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"

  schedule {
    frequency        = "daily"
    hour             = 3
    retention_seconds = 86400
  }
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_snapshot_schedule.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("neon_snapshot_schedule.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("neon_snapshot_schedule.test", "schedule.#", "1"),
					testutil.CheckResourceAttr("neon_snapshot_schedule.test", "schedule.0.frequency", "daily"),
					testutil.CheckResourceAttr("neon_snapshot_schedule.test", "schedule.0.hour", "3"),
					testutil.CheckResourceAttr("neon_snapshot_schedule.test", "schedule.0.retention_seconds", "86400"),
				),
			},
		},
	})
}

func TestSnapshotScheduleResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupSnapshotScheduleMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_snapshot_schedule" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"

  schedule {
    frequency        = "daily"
    hour             = 3
    retention_seconds = 86400
  }
}
`),
			},
			{
				ResourceName:                         "neon_snapshot_schedule.test",
				ImportState:                          true,
				ImportStateId:                        "test-project-id/br-test-001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "project_id",
			},
		},
	})
}

func TestSnapshotScheduleResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPut,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/backup_schedule",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_snapshot_schedule" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"

  schedule {
    frequency = "daily"
    hour      = 3
  }
}
`),
				ExpectError: regexp.MustCompile(`Failed to create snapshot schedule`),
			},
		},
	})
}
