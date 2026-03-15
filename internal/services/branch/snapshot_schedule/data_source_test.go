package snapshot_schedule_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestSnapshotScheduleDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-test-001/backup_schedule",
		testutil.JSONResponder(200, scheduleGetResponseJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_snapshot_schedule" "test" {
  project_id = "test-project-id"
  branch_id  = "br-test-001"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_snapshot_schedule.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("data.neon_snapshot_schedule.test", "branch_id", "br-test-001"),
					testutil.CheckResourceAttr("data.neon_snapshot_schedule.test", "schedule.#", "1"),
					testutil.CheckResourceAttr("data.neon_snapshot_schedule.test", "schedule.0.frequency", "daily"),
					testutil.CheckResourceAttr("data.neon_snapshot_schedule.test", "schedule.0.hour", "3"),
					testutil.CheckResourceAttr("data.neon_snapshot_schedule.test", "schedule.0.retention_seconds", "86400"),
				),
			},
		},
	})
}
