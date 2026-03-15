package data_api

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type branchDataAPIDataSource struct {
	client *neon.Client
}

type branchDataAPIDataSourceModel struct {
	ProjectID    types.String `tfsdk:"project_id"`
	BranchID     types.String `tfsdk:"branch_id"`
	DatabaseName types.String `tfsdk:"database_name"`
	URL          types.String `tfsdk:"url"`
	Status       types.String `tfsdk:"status"`
}

func NewDataSource() datasource.DataSource {
	return &branchDataAPIDataSource{}
}

func (d *branchDataAPIDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_data_api"
}

func (d *branchDataAPIDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a Neon branch Data API.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The Neon branch ID.",
				Required:    true,
			},
			"database_name": schema.StringAttribute{
				Description: "The database name.",
				Required:    true,
			},
			"url": schema.StringAttribute{
				Description: "The Data API URL.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The status of the Data API.",
				Computed:    true,
			},
		},
	}
}

func (d *branchDataAPIDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *branchDataAPIDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data branchDataAPIDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetProjectBranchDataAPI(ctx, neon.GetProjectBranchDataAPIParams{
		ProjectID:    data.ProjectID.ValueString(),
		BranchID:     data.BranchID.ValueString(),
		DatabaseName: data.DatabaseName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read branch data API", err.Error())
		return
	}

	data.URL = types.StringValue(formatURL(result.URL))
	data.Status = types.StringValue(result.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
