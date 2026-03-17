package jwks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ resource.Resource                = &jwksResource{}
	_ resource.ResourceWithConfigure   = &jwksResource{}
	_ resource.ResourceWithImportState = &jwksResource{}
)

type jwksResource struct {
	client *neon.Client
}

type jwksResourceModel struct {
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

func NewResource() resource.Resource {
	return &jwksResource{}
}

func (r *jwksResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_jwks"
}

func (r *jwksResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a JWKS URL for a Neon project, used for verifying JWTs as the authentication mechanism.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The JWKS ID.",
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
			"jwks_url": schema.StringAttribute{
				Description: "The URL that lists the JWKS.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"provider_name": schema.StringAttribute{
				Description: "The name of the authentication provider (e.g., Clerk, Stytch, Auth0).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID on which the JWKS URL will be accepted. If not specified, the JWKS URL will be accepted on all branches.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"jwt_audience": schema.StringAttribute{
				Description: "The name of the required JWT Audience to be used.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_names": schema.ListAttribute{
				Description: "The roles the JWKS should be mapped to. By default, the JWKS is mapped to the authenticator, authenticated and anonymous roles.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *jwksResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *jwksResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data jwksResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &neon.AddProjectJWKSRequest{
		JwksURL:      data.JwksURL.ValueString(),
		ProviderName: data.ProviderName.ValueString(),
	}

	if !data.BranchID.IsNull() && !data.BranchID.IsUnknown() {
		apiReq.BranchID = neon.NewOptString(data.BranchID.ValueString())
	}
	if !data.JwtAudience.IsNull() && !data.JwtAudience.IsUnknown() {
		apiReq.JwtAudience = neon.NewOptString(data.JwtAudience.ValueString())
	}
	if !data.RoleNames.IsNull() && !data.RoleNames.IsUnknown() {
		var roleNames []string
		resp.Diagnostics.Append(data.RoleNames.ElementsAs(ctx, &roleNames, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		apiReq.RoleNames = roleNames //nolint:staticcheck // intentionally using deprecated API field for backward compatibility
	}

	result, err := r.client.AddProjectJWKS(ctx, apiReq, neon.AddProjectJWKSParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create JWKS", err.Error())
		return
	}

	r.mapJWKSToModel(&result.Jwks, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *jwksResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data jwksResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetProjectJWKS(ctx, neon.GetProjectJWKSParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read JWKS", err.Error())
		return
	}

	for i := range result.Jwks {
		if result.Jwks[i].ID == data.ID.ValueString() {
			r.mapJWKSToModel(&result.Jwks[i], &data)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *jwksResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All attributes require replacement, so Update is never called.
}

func (r *jwksResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data jwksResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectJWKS(ctx, neon.DeleteProjectJWKSParams{
		ProjectID: data.ProjectID.ValueString(),
		JwksID:    data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete JWKS", err.Error())
		return
	}
}

func (r *jwksResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: {project_id}/{jwks_id}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *jwksResource) mapJWKSToModel(j *neon.JWKS, data *jwksResourceModel) {
	data.ID = types.StringValue(j.ID)
	data.ProjectID = types.StringValue(j.ProjectID)
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
		for i, name := range j.RoleNames {
			roleNameValues[i] = types.StringValue(name)
		}
		data.RoleNames, _ = types.ListValueFrom(context.Background(), types.StringType, roleNameValues)
	} else {
		data.RoleNames = types.ListNull(types.StringType)
	}

	data.CreatedAt = types.StringValue(j.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(j.UpdatedAt.Format(time.RFC3339))
}
