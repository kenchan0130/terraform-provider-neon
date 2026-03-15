package member

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonerror"
)

var (
	_ resource.Resource                = &organizationMemberRoleResource{}
	_ resource.ResourceWithConfigure   = &organizationMemberRoleResource{}
	_ resource.ResourceWithImportState = &organizationMemberRoleResource{}
)

type organizationMemberRoleResource struct {
	client *neon.Client
}

type organizationMemberRoleResourceModel struct {
	OrgID    types.String `tfsdk:"org_id"`
	MemberID types.String `tfsdk:"member_id"`
	Role     types.String `tfsdk:"role"`
	UserID   types.String `tfsdk:"user_id"`
	JoinedAt types.String `tfsdk:"joined_at"`
}

func NewResource() resource.Resource {
	return &organizationMemberRoleResource{}
}

func (r *organizationMemberRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_member_role"
}

func (r *organizationMemberRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the role of a Neon organization member. The member must already exist in the organization (e.g., via an accepted invitation). Destroying this resource only removes it from Terraform state; the member remains in the organization.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Description: "The Neon organization ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"member_id": schema.StringAttribute{
				Description: "The organization member ID (UUID).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Description: "The member role. Valid values are `admin` or `member`.",
				Required:    true,
			},
			"user_id": schema.StringAttribute{
				Description: "The user ID of the member.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"joined_at": schema.StringAttribute{
				Description: "A timestamp indicating when the member joined the organization, in RFC 3339 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *organizationMemberRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client
}

func (r *organizationMemberRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationMemberRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.createOrUpdate(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationMemberRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationMemberRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	memberUUID, err := uuid.Parse(data.MemberID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid member ID", fmt.Sprintf("Failed to parse member_id as UUID: %s", err))
		return
	}

	result, err := r.client.GetOrganizationMember(ctx, neon.GetOrganizationMemberParams{
		OrgID:    data.OrgID.ValueString(),
		MemberID: memberUUID,
	})
	if err != nil {
		if neonerror.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read organization member", err.Error())
		return
	}

	mapMemberToModel(result, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationMemberRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationMemberRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.createOrUpdate(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationMemberRoleResource) createOrUpdate(ctx context.Context, data *organizationMemberRoleResourceModel, diagnostics *diag.Diagnostics) {
	memberUUID, err := uuid.Parse(data.MemberID.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid member ID", fmt.Sprintf("Failed to parse member_id as UUID: %s", err))
		return
	}

	result, err := r.client.UpdateOrganizationMember(ctx, &neon.OrganizationMemberUpdateRequest{
		Role: neon.MemberRole(data.Role.ValueString()),
	}, neon.UpdateOrganizationMemberParams{
		OrgID:    data.OrgID.ValueString(),
		MemberID: memberUUID,
	})
	if err != nil {
		diagnostics.AddError("Failed to update organization member role", err.Error())
		return
	}

	mapMemberToModel(result, data)
}

func (r *organizationMemberRoleResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: destroying this resource only removes it from Terraform state.
	// The member remains in the organization.
}

func (r *organizationMemberRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: {org_id}/{member_id}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("member_id"), parts[1])...)
}

func mapMemberToModel(m *neon.Member, data *organizationMemberRoleResourceModel) {
	data.MemberID = types.StringValue(m.ID.String())
	data.OrgID = types.StringValue(m.OrgID)
	data.Role = types.StringValue(string(m.Role))
	data.UserID = types.StringValue(m.UserID.String())

	if m.JoinedAt.IsSet() {
		data.JoinedAt = types.StringValue(m.JoinedAt.Value.Format(time.RFC3339))
	} else {
		data.JoinedAt = types.StringNull()
	}
}
