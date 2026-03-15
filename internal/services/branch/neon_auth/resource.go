package neon_auth

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	_ resource.Resource                = &neonAuthResource{}
	_ resource.ResourceWithConfigure   = &neonAuthResource{}
	_ resource.ResourceWithImportState = &neonAuthResource{}
)

type neonAuthResource struct {
	client *neon.Client
}

type neonAuthResourceModel struct {
	ProjectID             types.String `tfsdk:"project_id"`
	BranchID              types.String `tfsdk:"branch_id"`
	AuthProvider          types.String `tfsdk:"auth_provider"`
	DatabaseName          types.String `tfsdk:"database_name"`
	AuthProviderProjectID types.String `tfsdk:"auth_provider_project_id"`
	DbName                types.String `tfsdk:"db_name"`
	JwksURL               types.String `tfsdk:"jwks_url"`
	BaseURL               types.String `tfsdk:"base_url"`
	CreatedAt             types.String `tfsdk:"created_at"`
}

func NewResource() resource.Resource {
	return &neonAuthResource{}
}

func (r *neonAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_neon_auth"
}

func (r *neonAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a NeonAuth integration on a branch.",
		Attributes: map[string]schema.Attribute{
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
			"auth_provider": schema.StringAttribute{
				Description: "The authentication provider (e.g. stack, stack_v2, better_auth).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"database_name": schema.StringAttribute{
				Description: "The database name for the NeonAuth integration.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"auth_provider_project_id": schema.StringAttribute{
				Description: "The auth provider project ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"db_name": schema.StringAttribute{
				Description: "The database name used by the integration.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"jwks_url": schema.StringAttribute{
				Description: "The JWKS URL for the NeonAuth integration.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"base_url": schema.StringAttribute{
				Description: "The base URL for the NeonAuth integration.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *neonAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *neonAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data neonAuthResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &neon.EnableNeonAuthIntegrationRequest{
		AuthProvider: neon.NeonAuthSupportedAuthProvider(data.AuthProvider.ValueString()),
	}
	if !data.DatabaseName.IsNull() && !data.DatabaseName.IsUnknown() {
		createReq.DatabaseName = neon.NewOptString(data.DatabaseName.ValueString())
	}

	result, err := r.client.CreateNeonAuth(ctx, createReq, neon.CreateNeonAuthParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create NeonAuth integration", err.Error())
		return
	}

	data.AuthProvider = types.StringValue(string(result.AuthProvider))
	data.AuthProviderProjectID = types.StringValue(result.AuthProviderProjectID)
	data.JwksURL = types.StringValue(result.JwksURL)

	if v, ok := result.BaseURL.Get(); ok {
		data.BaseURL = types.StringValue(v)
	} else {
		data.BaseURL = types.StringNull()
	}

	// Read back to get full state including db_name and created_at
	integration, err := r.client.GetNeonAuth(ctx, neon.GetNeonAuthParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read NeonAuth integration after create", err.Error())
		return
	}

	mapNeonAuthIntegrationToModel(integration, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *neonAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data neonAuthResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetNeonAuth(ctx, neon.GetNeonAuthParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		if neonerror.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read NeonAuth integration", err.Error())
		return
	}

	mapNeonAuthIntegrationToModel(result, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *neonAuthResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"NeonAuth integration does not support updates. All attributes require replacement.",
	)
}

func (r *neonAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data neonAuthResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DisableNeonAuth(ctx, neon.OptDisableNeonAuthReq{}, neon.DisableNeonAuthParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to disable NeonAuth integration", err.Error())
		return
	}
}

func (r *neonAuthResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: {project_id}/{branch_id}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("branch_id"), parts[1])...)
}

func mapNeonAuthIntegrationToModel(integration *neon.NeonAuthIntegration, data *neonAuthResourceModel) {
	data.AuthProvider = types.StringValue(string(integration.AuthProvider))
	data.AuthProviderProjectID = types.StringValue(integration.AuthProviderProjectID)
	data.DbName = types.StringValue(integration.DbName)
	data.JwksURL = types.StringValue(integration.JwksURL)
	data.CreatedAt = types.StringValue(integration.CreatedAt.Format(time.RFC3339))

	if v, ok := integration.BaseURL.Get(); ok {
		data.BaseURL = types.StringValue(v)
	} else {
		data.BaseURL = types.StringNull()
	}
}
