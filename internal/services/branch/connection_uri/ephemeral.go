package connection_uri //nolint:dupl

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ ephemeral.EphemeralResource              = &connectionURIEphemeral{}
	_ ephemeral.EphemeralResourceWithConfigure = &connectionURIEphemeral{}
)

type connectionURIEphemeral struct {
	client *neon.Client
}

func NewEphemeralResource() ephemeral.EphemeralResource {
	return &connectionURIEphemeral{}
}

func (e *connectionURIEphemeral) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connection_uri"
}

func (e *connectionURIEphemeral) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a connection URI for a Neon database. The URI is ephemeral and will not be stored in Terraform state.",
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
				Sensitive:   true,
			},
		},
	}
}

func (e *connectionURIEphemeral) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (e *connectionURIEphemeral) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data connectionURIModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(fetchConnectionURI(ctx, e.client, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
