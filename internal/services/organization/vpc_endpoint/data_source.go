package vpc_endpoint

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type vpcEndpointDataSource struct {
	client *neon.Client
}

type vpcEndpointDataSourceModel struct {
	OrgID                     types.String `tfsdk:"org_id"`
	RegionID                  types.String `tfsdk:"region_id"`
	VpcEndpointID             types.String `tfsdk:"vpc_endpoint_id"`
	Label                     types.String `tfsdk:"label"`
	State                     types.String `tfsdk:"state"`
	NumRestrictedProjects     types.Int64  `tfsdk:"num_restricted_projects"`
	ExampleRestrictedProjects types.List   `tfsdk:"example_restricted_projects"`
}

func NewDataSource() datasource.DataSource {
	return &vpcEndpointDataSource{}
}

func (d *vpcEndpointDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_vpc_endpoint"
}

func (d *vpcEndpointDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a VPC endpoint assigned to a Neon organization.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Description: "The Neon organization ID.",
				Required:    true,
			},
			"region_id": schema.StringAttribute{
				Description: "The Neon region ID.",
				Required:    true,
			},
			"vpc_endpoint_id": schema.StringAttribute{
				Description: "The VPC endpoint ID.",
				Required:    true,
			},
			"label": schema.StringAttribute{
				Description: "A descriptive label for the VPC endpoint.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "The current state of the VPC endpoint. Possible values are `new` or `accepted`.",
				Computed:    true,
			},
			"num_restricted_projects": schema.Int64Attribute{
				Description: "The number of projects that are restricted to use this VPC endpoint.",
				Computed:    true,
			},
			"example_restricted_projects": schema.ListAttribute{
				Description: "A list of example projects that are restricted to use this VPC endpoint (at most 3).",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *vpcEndpointDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *vpcEndpointDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data vpcEndpointDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetOrganizationVPCEndpointDetails(ctx, neon.GetOrganizationVPCEndpointDetailsParams{
		OrgID:         data.OrgID.ValueString(),
		RegionID:      data.RegionID.ValueString(),
		VpcEndpointID: data.VpcEndpointID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read VPC endpoint details", err.Error())
		return
	}

	var rm vpcEndpointResourceModel
	mapVPCEndpointToModel(result, &rm, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data.VpcEndpointID = rm.VpcEndpointID
	data.Label = rm.Label
	data.State = rm.State
	data.NumRestrictedProjects = rm.NumRestrictedProjects
	data.ExampleRestrictedProjects = rm.ExampleRestrictedProjects

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
