package project_access

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ resource.Resource                = &projectAccessResource{}
	_ resource.ResourceWithConfigure   = &projectAccessResource{}
	_ resource.ResourceWithImportState = &projectAccessResource{}
)

type projectAccessResource struct {
	client *neon.Client
}

type projectAccessResourceModel struct {
	ProjectID      types.String `tfsdk:"project_id"`
	PermissionID   types.String `tfsdk:"permission_id"`
	GrantedToEmail types.String `tfsdk:"granted_to_email"`
	GrantedAt      types.String `tfsdk:"granted_at"`
}

func NewResource() resource.Resource {
	return &projectAccessResource{}
}

func (r *projectAccessResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_access"
}

func (r *projectAccessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Grants access to a Neon project by sharing it with a specified email address.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permission_id": schema.StringAttribute{
				Description: "The permission ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"granted_to_email": schema.StringAttribute{
				Description: "The email address to grant project access to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"granted_at": schema.StringAttribute{
				Description: "The timestamp when the permission was granted.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *projectAccessResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *projectAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data projectAccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GrantPermissionToProject(ctx, &neon.GrantPermissionToProjectRequest{
		Email: data.GrantedToEmail.ValueString(),
	}, neon.GrantPermissionToProjectParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to grant project permission", err.Error())
		return
	}

	data.PermissionID = types.StringValue(result.ID)
	data.GrantedToEmail = types.StringValue(result.GrantedToEmail)
	data.GrantedAt = types.StringValue(result.GrantedAt.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *projectAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data projectAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ListProjectPermissions(ctx, neon.ListProjectPermissionsParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list project permissions", err.Error())
		return
	}

	for _, perm := range result.ProjectPermissions {
		if perm.ID == data.PermissionID.ValueString() {
			if perm.RevokedAt.IsSet() {
				resp.State.RemoveResource(ctx)
				return
			}
			data.GrantedToEmail = types.StringValue(perm.GrantedToEmail)
			data.GrantedAt = types.StringValue(perm.GrantedAt.String())
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *projectAccessResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All attributes require replacement, so Update is never called.
}

func (r *projectAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data projectAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokePermissionFromProject(ctx, neon.RevokePermissionFromProjectParams{
		ProjectID:    data.ProjectID.ValueString(),
		PermissionID: data.PermissionID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to revoke project permission", err.Error())
		return
	}
}

func (r *projectAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format: project_id/permission_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission_id"), parts[1])...)
}
