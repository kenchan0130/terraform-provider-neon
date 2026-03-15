package database

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type databaseDataSource struct {
	client *neon.Client
}

type databaseDataSourceModel struct {
	ID        types.Int64  `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	BranchID  types.String `tfsdk:"branch_id"`
	Name      types.String `tfsdk:"name"`
	OwnerName types.String `tfsdk:"owner_name"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewDataSource() datasource.DataSource {
	return &databaseDataSource{}
}

func (d *databaseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (d *databaseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a Neon database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The database ID.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The database name.",
				Required:    true,
			},
			"owner_name": schema.StringAttribute{
				Description: "The name of the role that owns the database.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp.",
				Computed:    true,
			},
		},
	}
}

func (d *databaseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *databaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data databaseDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetProjectBranchDatabase(ctx, neon.GetProjectBranchDatabaseParams{
		ProjectID:    data.ProjectID.ValueString(),
		BranchID:     data.BranchID.ValueString(),
		DatabaseName: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read database", err.Error())
		return
	}

	db := &result.Database
	data.ID = types.Int64Value(db.ID)
	data.BranchID = types.StringValue(db.BranchID)
	data.Name = types.StringValue(db.Name)
	data.OwnerName = types.StringValue(db.OwnerName)
	data.CreatedAt = types.StringValue(db.CreatedAt.String())
	data.UpdatedAt = types.StringValue(db.UpdatedAt.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
