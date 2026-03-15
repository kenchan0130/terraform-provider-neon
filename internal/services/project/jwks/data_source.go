package jwks

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type jwksDataSource struct {
	client *neon.Client
}

type jwksDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	ProjectID    types.String `tfsdk:"project_id"`
	JwksURL      types.String `tfsdk:"jwks_url"`
	ProviderName types.String `tfsdk:"provider_name"`
	BranchID     types.String `tfsdk:"branch_id"`
	JwtAudience  types.String `tfsdk:"jwt_audience"`
	RoleNames    types.List   `tfsdk:"role_names"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func NewDataSource() datasource.DataSource {
	return &jwksDataSource{}
}

func (d *jwksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_jwks"
}

func (d *jwksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a JWKS URL configured for a Neon project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The JWKS ID.",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"jwks_url": schema.StringAttribute{
				Description: "The URL that lists the JWKS.",
				Computed:    true,
			},
			"provider_name": schema.StringAttribute{
				Description: "The name of the authentication provider (e.g., Clerk, Stytch, Auth0).",
				Computed:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID on which the JWKS URL will be accepted.",
				Computed:    true,
			},
			"jwt_audience": schema.StringAttribute{
				Description: "The name of the required JWT Audience to be used.",
				Computed:    true,
			},
			"role_names": schema.ListAttribute{
				Description: "The roles the JWKS is mapped to.",
				Computed:    true,
				ElementType: types.StringType,
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

func (d *jwksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *jwksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data jwksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetProjectJWKS(ctx, neon.GetProjectJWKSParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read JWKS", err.Error())
		return
	}

	for i := range result.Jwks {
		if result.Jwks[i].ID == data.ID.ValueString() {
			j := &result.Jwks[i]
			data.JwksURL = types.StringValue(j.JwksURL)
			data.ProviderName = types.StringValue(j.ProviderName)

			if v, ok := j.BranchID.Get(); ok {
				data.BranchID = types.StringValue(v)
			} else {
				data.BranchID = types.StringNull()
			}

			if v, ok := j.JwtAudience.Get(); ok {
				data.JwtAudience = types.StringValue(v)
			} else {
				data.JwtAudience = types.StringNull()
			}

			if len(j.RoleNames) > 0 {
				roleNameValues := make([]types.String, len(j.RoleNames))
				for k, name := range j.RoleNames {
					roleNameValues[k] = types.StringValue(name)
				}
				data.RoleNames, _ = types.ListValueFrom(ctx, types.StringType, roleNameValues)
			} else {
				data.RoleNames = types.ListNull(types.StringType)
			}

			data.CreatedAt = types.StringValue(j.CreatedAt.Format(time.RFC3339))
			data.UpdatedAt = types.StringValue(j.UpdatedAt.Format(time.RFC3339))

			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"JWKS Not Found",
		fmt.Sprintf("JWKS with ID %q not found in project %q.", data.ID.ValueString(), data.ProjectID.ValueString()),
	)
}
