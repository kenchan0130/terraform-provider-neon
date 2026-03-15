package neon_auth

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type neonAuthDataSource struct {
	client *neon.Client
}

type neonAuthDataSourceModel struct {
	ProjectID             types.String `tfsdk:"project_id"`
	BranchID              types.String `tfsdk:"branch_id"`
	AuthProvider          types.String `tfsdk:"auth_provider"`
	AuthProviderProjectID types.String `tfsdk:"auth_provider_project_id"`
	DbName                types.String `tfsdk:"db_name"`
	JwksURL               types.String `tfsdk:"jwks_url"`
	BaseURL               types.String `tfsdk:"base_url"`
	CreatedAt             types.String `tfsdk:"created_at"`
}

func NewDataSource() datasource.DataSource {
	return &neonAuthDataSource{}
}

func (d *neonAuthDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_neon_auth"
}

func (d *neonAuthDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a NeonAuth integration on a branch.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The Neon branch ID.",
				Required:    true,
			},
			"auth_provider": schema.StringAttribute{
				Description: "The authentication provider.",
				Computed:    true,
			},
			"auth_provider_project_id": schema.StringAttribute{
				Description: "The auth provider project ID.",
				Computed:    true,
			},
			"db_name": schema.StringAttribute{
				Description: "The database name used by the integration.",
				Computed:    true,
			},
			"jwks_url": schema.StringAttribute{
				Description: "The JWKS URL for the NeonAuth integration.",
				Computed:    true,
			},
			"base_url": schema.StringAttribute{
				Description: "The base URL for the NeonAuth integration.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
			},
		},
	}
}

func (d *neonAuthDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *neonAuthDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data neonAuthDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetNeonAuth(ctx, neon.GetNeonAuthParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read NeonAuth integration", err.Error())
		return
	}

	data.AuthProvider = types.StringValue(string(result.AuthProvider))
	data.AuthProviderProjectID = types.StringValue(result.AuthProviderProjectID)
	data.DbName = types.StringValue(result.DbName)
	data.JwksURL = types.StringValue(result.JwksURL)
	data.CreatedAt = types.StringValue(result.CreatedAt.Format(time.RFC3339))

	if v, ok := result.BaseURL.Get(); ok {
		data.BaseURL = types.StringValue(v)
	} else {
		data.BaseURL = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
