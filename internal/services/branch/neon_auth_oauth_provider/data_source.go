package neon_auth_oauth_provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type neonAuthOauthProviderDataSource struct {
	client *neon.Client
}

func NewDataSource() datasource.DataSource {
	return &neonAuthOauthProviderDataSource{}
}

func (d *neonAuthOauthProviderDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_neon_auth_oauth_provider"
}

func (d *neonAuthOauthProviderDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a NeonAuth OAuth provider on a branch.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The OAuth provider ID (e.g. `google`, `github`, `microsoft`, `vercel`).",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The Neon branch ID.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The OAuth provider type (e.g. `standard`, `shared`).",
				Computed:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The OAuth client ID.",
				Computed:    true,
			},
			"client_secret": schema.StringAttribute{
				Description: "The OAuth client secret.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (d *neonAuthOauthProviderDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *neonAuthOauthProviderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data oauthProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(fetchOauthProvider(ctx, d.client, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
