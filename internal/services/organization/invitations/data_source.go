package invitations

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type organizationInvitationsDataSource struct {
	client *neon.Client
}

type organizationInvitationsDataSourceModel struct {
	OrgID       types.String      `tfsdk:"org_id"`
	Invitations []invitationModel `tfsdk:"invitations"`
}

type invitationModel struct {
	ID        types.String `tfsdk:"id"`
	Email     types.String `tfsdk:"email"`
	OrgID     types.String `tfsdk:"org_id"`
	InvitedBy types.String `tfsdk:"invited_by"`
	InvitedAt types.String `tfsdk:"invited_at"`
	Role      types.String `tfsdk:"role"`
}

func NewDataSource() datasource.DataSource {
	return &organizationInvitationsDataSource{}
}

func (d *organizationInvitationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_invitations"
}

func (d *organizationInvitationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the list of invitations for a Neon organization.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Description: "The Neon organization ID.",
				Required:    true,
			},
			"invitations": schema.ListNestedAttribute{
				Description: "The list of organization invitations.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The invitation ID.",
							Computed:    true,
						},
						"email": schema.StringAttribute{
							Description: "The email of the invited user.",
							Computed:    true,
						},
						"org_id": schema.StringAttribute{
							Description: "The organization ID.",
							Computed:    true,
						},
						"invited_by": schema.StringAttribute{
							Description: "The user ID of the person who sent the invitation.",
							Computed:    true,
						},
						"invited_at": schema.StringAttribute{
							Description: "A timestamp indicating when the invitation was created, in RFC 3339 format.",
							Computed:    true,
						},
						"role": schema.StringAttribute{
							Description: "The role assigned to the invited user.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *organizationInvitationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *organizationInvitationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationInvitationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetOrganizationInvitations(ctx, neon.GetOrganizationInvitationsParams{
		OrgID: data.OrgID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read organization invitations", err.Error())
		return
	}

	data.Invitations = make([]invitationModel, len(result.Invitations))
	for i, inv := range result.Invitations {
		data.Invitations[i] = invitationModel{
			ID:        types.StringValue(inv.ID.String()),
			Email:     types.StringValue(inv.Email),
			OrgID:     types.StringValue(inv.OrgID),
			InvitedBy: types.StringValue(inv.InvitedBy.String()),
			InvitedAt: types.StringValue(inv.InvitedAt.Format(time.RFC3339)),
			Role:      types.StringValue(string(inv.Role)),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
