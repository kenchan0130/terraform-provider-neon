package neon_auth_oauth_provider

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
	_ resource.Resource                = &neonAuthOauthProviderResource{}
	_ resource.ResourceWithConfigure   = &neonAuthOauthProviderResource{}
	_ resource.ResourceWithImportState = &neonAuthOauthProviderResource{}
)

type neonAuthOauthProviderResource struct {
	client *neon.Client
}

type neonAuthOauthProviderResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ProjectID    types.String `tfsdk:"project_id"`
	BranchID     types.String `tfsdk:"branch_id"`
	Type         types.String `tfsdk:"type"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

func NewResource() resource.Resource {
	return &neonAuthOauthProviderResource{}
}

func (r *neonAuthOauthProviderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_neon_auth_oauth_provider"
}

func (r *neonAuthOauthProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a NeonAuth OAuth provider on a branch.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The OAuth provider ID (e.g. google, github, microsoft, vercel).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"branch_id": schema.StringAttribute{
				Description: "The Neon branch ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The OAuth provider type (e.g. standard, shared).",
				Required:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The OAuth client ID.",
				Optional:    true,
				Sensitive:   true,
			},
			"client_secret": schema.StringAttribute{
				Description: "The OAuth client secret.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *neonAuthOauthProviderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *neonAuthOauthProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data neonAuthOauthProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &neon.NeonAuthAddOAuthProviderRequest{
		ID: neon.NeonAuthOauthProviderId(data.Type.ValueString()),
	}
	if !data.ClientID.IsNull() && !data.ClientID.IsUnknown() {
		createReq.ClientID = neon.NewOptString(data.ClientID.ValueString())
	}
	if !data.ClientSecret.IsNull() && !data.ClientSecret.IsUnknown() {
		createReq.ClientSecret = neon.NewOptString(data.ClientSecret.ValueString())
	}

	result, err := r.client.AddBranchNeonAuthOauthProvider(ctx, createReq, neon.AddBranchNeonAuthOauthProviderParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to add NeonAuth OAuth provider", err.Error())
		return
	}

	mapNeonAuthOauthProviderToModel(result, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *neonAuthOauthProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data neonAuthOauthProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ListBranchNeonAuthOauthProviders(ctx, neon.ListBranchNeonAuthOauthProvidersParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list NeonAuth OAuth providers", err.Error())
		return
	}

	for i := range result.Providers {
		if string(result.Providers[i].ID) == data.ID.ValueString() {
			mapNeonAuthOauthProviderToModel(&result.Providers[i], &data)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *neonAuthOauthProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan neonAuthOauthProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state neonAuthOauthProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &neon.NeonAuthUpdateOAuthProviderRequest{}
	if !plan.ClientID.IsNull() && !plan.ClientID.IsUnknown() {
		updateReq.ClientID = neon.NewOptString(plan.ClientID.ValueString())
	}
	if !plan.ClientSecret.IsNull() && !plan.ClientSecret.IsUnknown() {
		updateReq.ClientSecret = neon.NewOptString(plan.ClientSecret.ValueString())
	}

	result, err := r.client.UpdateBranchNeonAuthOauthProvider(ctx, updateReq, neon.UpdateBranchNeonAuthOauthProviderParams{
		ProjectID:       state.ProjectID.ValueString(),
		BranchID:        state.BranchID.ValueString(),
		OAuthProviderID: neon.NeonAuthOauthProviderId(state.ID.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update NeonAuth OAuth provider", err.Error())
		return
	}

	mapNeonAuthOauthProviderToModel(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *neonAuthOauthProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data neonAuthOauthProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBranchNeonAuthOauthProvider(ctx, neon.DeleteBranchNeonAuthOauthProviderParams{
		ProjectID:       data.ProjectID.ValueString(),
		BranchID:        data.BranchID.ValueString(),
		OAuthProviderID: neon.NeonAuthOauthProviderId(data.ID.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete NeonAuth OAuth provider", err.Error())
		return
	}
}

func (r *neonAuthOauthProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: {project_id}/{branch_id}/{provider_id}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("branch_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func mapNeonAuthOauthProviderToModel(provider *neon.NeonAuthOauthProvider, data *neonAuthOauthProviderResourceModel) {
	data.ID = types.StringValue(string(provider.ID))
	data.Type = types.StringValue(string(provider.Type))

	if v, ok := provider.ClientID.Get(); ok {
		data.ClientID = types.StringValue(v)
	} else {
		data.ClientID = types.StringNull()
	}

	if v, ok := provider.ClientSecret.Get(); ok {
		data.ClientSecret = types.StringValue(v)
	} else {
		data.ClientSecret = types.StringNull()
	}
}
