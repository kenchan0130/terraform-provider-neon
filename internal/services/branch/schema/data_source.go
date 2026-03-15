package branch_schema

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type branchSchemaDataSource struct {
	client *neon.Client
}

type branchSchemaDataSourceModel struct {
	ProjectID    types.String `tfsdk:"project_id"`
	BranchID     types.String `tfsdk:"branch_id"`
	DatabaseName types.String `tfsdk:"database_name"`
	SQL          types.String `tfsdk:"sql"`
}

func NewDataSource() datasource.DataSource {
	return &branchSchemaDataSource{}
}

func (d *branchSchemaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_schema"
}

func (d *branchSchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the schema for a Neon branch database in SQL format.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID.",
				Required:    true,
			},
			"database_name": schema.StringAttribute{
				Description: "The name of the database for which the schema is retrieved.",
				Required:    true,
			},
			"sql": schema.StringAttribute{
				Description: "The database schema in SQL format.",
				Computed:    true,
			},
		},
	}
}

func (d *branchSchemaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *branchSchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data branchSchemaDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetProjectBranchSchema(ctx, neon.GetProjectBranchSchemaParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
		DbName:    data.DatabaseName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read branch schema", err.Error())
		return
	}

	if result.SQL.IsSet() {
		data.SQL = types.StringValue(result.SQL.Value)
	} else {
		data.SQL = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
