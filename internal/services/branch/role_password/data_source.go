package role_password //nolint:dupl

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ datasource.DataSource              = &rolePasswordDataSource{}
	_ datasource.DataSourceWithConfigure = &rolePasswordDataSource{}
)

type rolePasswordDataSource struct {
	client *neon.Client
}

func NewDataSource() datasource.DataSource {
	return &rolePasswordDataSource{}
}

func (d *rolePasswordDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_password"
}

func (d *rolePasswordDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the password for a Neon role.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID.",
				Required:    true,
			},
			"role_name": schema.StringAttribute{
				Description: "The role name.",
				Required:    true,
			},
			"password": schema.StringAttribute{
				Description: "The role password.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (d *rolePasswordDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *rolePasswordDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data rolePasswordModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(fetchRolePassword(ctx, d.client, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
