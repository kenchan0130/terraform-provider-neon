package vpc_endpoint_restriction

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type vpcEndpointRestrictionDataSource struct {
	client *neon.Client
}

type vpcEndpointRestrictionDataSourceModel struct {
	ProjectID     types.String `tfsdk:"project_id"`
	VpcEndpointID types.String `tfsdk:"vpc_endpoint_id"`
	Label         types.String `tfsdk:"label"`
}

func NewDataSource() datasource.DataSource {
	return &vpcEndpointRestrictionDataSource{}
}

func (d *vpcEndpointRestrictionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_vpc_endpoint_restriction"
}

func (d *vpcEndpointRestrictionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a VPC endpoint restriction for a Neon project.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
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
		},
	}
}

func (d *vpcEndpointRestrictionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *vpcEndpointRestrictionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data vpcEndpointRestrictionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.ListProjectVPCEndpoints(ctx, neon.ListProjectVPCEndpointsParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list VPC endpoint restrictions", err.Error())
		return
	}

	for _, ep := range result.Endpoints {
		if ep.VpcEndpointID == data.VpcEndpointID.ValueString() {
			data.Label = types.StringValue(ep.Label)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"VPC Endpoint Restriction Not Found",
		fmt.Sprintf("VPC endpoint %q not found in project %q.", data.VpcEndpointID.ValueString(), data.ProjectID.ValueString()),
	)
}
