package connection_uri

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type connectionURIDataSource struct {
	client *neon.Client
}

type connectionURIDataSourceModel struct {
	ProjectID    types.String `tfsdk:"project_id"`
	BranchID     types.String `tfsdk:"branch_id"`
	EndpointID   types.String `tfsdk:"endpoint_id"`
	DatabaseName types.String `tfsdk:"database_name"`
	RoleName     types.String `tfsdk:"role_name"`
	Pooled       types.Bool   `tfsdk:"pooled"`
	URI          types.String `tfsdk:"uri"`
}

func NewDataSource() datasource.DataSource {
	return &connectionURIDataSource{}
}

func (d *connectionURIDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connection_uri"
}

func (d *connectionURIDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a connection URI for a Neon database.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID. Defaults to the project's default branch if not specified.",
				Optional:    true,
			},
			"endpoint_id": schema.StringAttribute{
				Description: "The endpoint ID. Defaults to the read-write endpoint associated with the branch if not specified.",
				Optional:    true,
			},
			"database_name": schema.StringAttribute{
				Description: "The database name.",
				Required:    true,
			},
			"role_name": schema.StringAttribute{
				Description: "The role name.",
				Required:    true,
			},
			"pooled": schema.BoolAttribute{
				Description: "Whether to use a pooled connection URI.",
				Optional:    true,
			},
			"uri": schema.StringAttribute{
				Description: "The connection URI.",
				Computed:    true,
			},
		},
	}
}

func (d *connectionURIDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *connectionURIDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data connectionURIDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := neon.GetConnectionURIParams{
		ProjectID:    data.ProjectID.ValueString(),
		DatabaseName: data.DatabaseName.ValueString(),
		RoleName:     data.RoleName.ValueString(),
	}

	if !data.BranchID.IsNull() {
		params.BranchID = neon.NewOptString(data.BranchID.ValueString())
	}

	if !data.EndpointID.IsNull() {
		params.EndpointID = neon.NewOptString(data.EndpointID.ValueString())
	}

	if !data.Pooled.IsNull() {
		params.Pooled = neon.NewOptBool(data.Pooled.ValueBool())
	}

	result, err := d.client.GetConnectionURI(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read connection URI", err.Error())
		return
	}

	data.URI = types.StringValue(result.URI)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
