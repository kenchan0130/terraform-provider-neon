package region

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type activeRegionsDataSource struct {
	client *neon.Client
}

type activeRegionsDataSourceModel struct {
	Regions []regionModel `tfsdk:"regions"`
}

type regionModel struct {
	RegionID types.String `tfsdk:"region_id"`
	Name     types.String `tfsdk:"name"`
	Default  types.Bool   `tfsdk:"default"`
	GeoLat   types.String `tfsdk:"geo_lat"`
	GeoLong  types.String `tfsdk:"geo_long"`
}

func NewDataSource() datasource.DataSource {
	return &activeRegionsDataSource{}
}

func (d *activeRegionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_active_regions"
}

func (d *activeRegionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the list of active regions available in Neon.",
		Attributes: map[string]schema.Attribute{
			"regions": schema.ListNestedAttribute{
				Description: "The list of active regions.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"region_id": schema.StringAttribute{
							Description: "The region ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "A short description of the region.",
							Computed:    true,
						},
						"default": schema.BoolAttribute{
							Description: "Whether this region is used by default in new projects.",
							Computed:    true,
						},
						"geo_lat": schema.StringAttribute{
							Description: "The geographical latitude (approximate) for the region.",
							Computed:    true,
						},
						"geo_long": schema.StringAttribute{
							Description: "The geographical longitude (approximate) for the region.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *activeRegionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*neon.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *neon.Client, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *activeRegionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data activeRegionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetActiveRegions(ctx, neon.GetActiveRegionsParams{})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read active regions", err.Error())
		return
	}

	data.Regions = make([]regionModel, len(result.Regions))
	for i, r := range result.Regions {
		data.Regions[i] = regionModel{
			RegionID: types.StringValue(r.RegionID),
			Name:     types.StringValue(r.Name),
			Default:  types.BoolValue(r.Default),
			GeoLat:   types.StringValue(r.GeoLat),
			GeoLong:  types.StringValue(r.GeoLong),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
