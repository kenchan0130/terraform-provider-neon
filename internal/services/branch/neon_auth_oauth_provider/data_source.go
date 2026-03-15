package neon_auth_oauth_provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type neonAuthOauthProviderDataSource struct {
	client *neon.Client
}

type neonAuthOauthProviderDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	ProjectID    types.String `tfsdk:"project_id"`
	BranchID     types.String `tfsdk:"branch_id"`
	Type         types.String `tfsdk:"type"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
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
				Description: "The OAuth provider ID (e.g. google, github, microsoft, vercel).",
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
				Description: "The OAuth provider type (e.g. standard, shared).",
				Computed:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The OAuth client ID.",
				Computed:    true,
				Sensitive:   true,
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
	var data neonAuthOauthProviderDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.ListBranchNeonAuthOauthProviders(ctx, neon.ListBranchNeonAuthOauthProvidersParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list NeonAuth OAuth providers", err.Error())
		return
	}

	for i := range result.Providers {
		if string(result.Providers[i].ID) == data.ID.ValueString() {
			p := &result.Providers[i]
			data.Type = types.StringValue(string(p.Type))

			if v, ok := p.ClientID.Get(); ok {
				data.ClientID = types.StringValue(v)
			} else {
				data.ClientID = types.StringNull()
			}

			if v, ok := p.ClientSecret.Get(); ok {
				data.ClientSecret = types.StringValue(v)
			} else {
				data.ClientSecret = types.StringNull()
			}

			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"OAuth Provider Not Found",
		fmt.Sprintf("OAuth provider %q not found for branch %q in project %q.", data.ID.ValueString(), data.BranchID.ValueString(), data.ProjectID.ValueString()),
	)
}
