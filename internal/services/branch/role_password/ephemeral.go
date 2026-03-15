package role_password //nolint:dupl

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ ephemeral.EphemeralResource              = &rolePasswordEphemeral{}
	_ ephemeral.EphemeralResourceWithConfigure = &rolePasswordEphemeral{}
)

type rolePasswordEphemeral struct {
	client *neon.Client
}

func NewEphemeralResource() ephemeral.EphemeralResource {
	return &rolePasswordEphemeral{}
}

func (e *rolePasswordEphemeral) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_password"
}

func (e *rolePasswordEphemeral) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the password for a Neon role. The password is ephemeral and will not be stored in Terraform state.",
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

func (e *rolePasswordEphemeral) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (e *rolePasswordEphemeral) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data rolePasswordModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(fetchRolePassword(ctx, e.client, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
