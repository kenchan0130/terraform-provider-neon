package snapshot_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func TestSnapshotDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/snapshots",
		testutil.JSONResponder(200, `{"snapshots": [`+snapshotJSON+`]}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_snapshot" "test" {
  project_id = "test-project-id"
  id         = "snap-test-001"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_snapshot.test", "id", "snap-test-001"),
					testutil.CheckResourceAttr("data.neon_snapshot.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("data.neon_snapshot.test", "name", "my-snapshot"),
					testutil.CheckResourceAttr("data.neon_snapshot.test", "source_branch_id", "br-test-001"),
					testutil.CheckResourceAttr("data.neon_snapshot.test", "manual", "true"),
				),
			},
		},
	})
}
