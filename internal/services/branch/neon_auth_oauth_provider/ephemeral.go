package neon_auth_oauth_provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ ephemeral.EphemeralResource              = &neonAuthOauthProviderEphemeral{}
	_ ephemeral.EphemeralResourceWithConfigure = &neonAuthOauthProviderEphemeral{}
)

type neonAuthOauthProviderEphemeral struct {
	client *neon.Client
}

func NewEphemeralResource() ephemeral.EphemeralResource {
	return &neonAuthOauthProviderEphemeral{}
}

func (e *neonAuthOauthProviderEphemeral) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_neon_auth_oauth_provider"
}

func (e *neonAuthOauthProviderEphemeral) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a NeonAuth OAuth provider on a branch. The sensitive fields (client_id, client_secret) are ephemeral and will not be stored in Terraform state.",
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

func (e *neonAuthOauthProviderEphemeral) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

	e.client = client
}

func (e *neonAuthOauthProviderEphemeral) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data oauthProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(fetchOauthProvider(ctx, e.client, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
