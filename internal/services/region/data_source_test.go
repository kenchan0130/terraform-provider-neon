package region_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const activeRegionsJSON = `{
	"regions": [
		{
			"region_id": "aws-us-east-1",
			"name": "US East (N. Virginia)",
			"default": true,
			"geo_lat": "39.0438",
			"geo_long": "-77.4874"
		},
		{
			"region_id": "aws-eu-west-1",
			"name": "Europe (Ireland)",
			"default": false,
			"geo_lat": "53.3331",
			"geo_long": "-6.2489"
		}
	]
}`

func TestActiveRegionsDataSource_Read(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/regions",
		testutil.JSONResponder(200, activeRegionsJSON),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
data "neon_active_regions" "test" {
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("data.neon_active_regions.test", "regions.#", "2"),
					testutil.CheckResourceAttr("data.neon_active_regions.test", "regions.0.region_id", "aws-us-east-1"),
					testutil.CheckResourceAttr("data.neon_active_regions.test", "regions.0.name", "US East (N. Virginia)"),
					testutil.CheckResourceAttr("data.neon_active_regions.test", "regions.0.default", "true"),
					testutil.CheckResourceAttr("data.neon_active_regions.test", "regions.0.geo_lat", "39.0438"),
					testutil.CheckResourceAttr("data.neon_active_regions.test", "regions.0.geo_long", "-77.4874"),
					testutil.CheckResourceAttr("data.neon_active_regions.test", "regions.1.region_id", "aws-eu-west-1"),
					testutil.CheckResourceAttr("data.neon_active_regions.test", "regions.1.name", "Europe (Ireland)"),
					testutil.CheckResourceAttr("data.neon_active_regions.test", "regions.1.default", "false"),
				),
			},
		},
	})
}
