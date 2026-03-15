package org_api_key

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ resource.Resource              = &organizationApiKeyResource{}
	_ resource.ResourceWithConfigure = &organizationApiKeyResource{}
)

type organizationApiKeyResource struct {
	client *neon.Client
}

type organizationApiKeyResourceModel struct {
	ID        types.Int64  `tfsdk:"id"`
	OrgID     types.String `tfsdk:"org_id"`
	Name      types.String `tfsdk:"name"`
	ProjectID types.String `tfsdk:"project_id"`
	Key       types.String `tfsdk:"key"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewResource() resource.Resource {
	return &organizationApiKeyResource{}
}

func (r *organizationApiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_api_key"
}

func (r *organizationApiKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an organization-scoped Neon API key. Import is not supported because the API key secret is only available at creation time.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The API key ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Description: "The Neon organization ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "A user-specified API key name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "If set, the API key can access only this project.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Description: "The generated API key. Only available after creation.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "A timestamp indicating when the API key was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *organizationApiKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *organizationApiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationApiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &neon.OrgApiKeyCreateRequest{
		KeyName: data.Name.ValueString(),
	}
	if !data.ProjectID.IsNull() && !data.ProjectID.IsUnknown() {
		createReq.ProjectID = neon.NewOptString(data.ProjectID.ValueString())
	}

	result, err := r.client.CreateOrgApiKey(ctx, createReq, neon.CreateOrgApiKeyParams{
		OrgID: data.OrgID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create organization API key", err.Error())
		return
	}

	data.ID = types.Int64Value(result.ID)
	data.Name = types.StringValue(result.Name)
	data.Key = types.StringValue(result.Key)
	data.CreatedAt = types.StringValue(result.CreatedAt.String())

	if v, ok := result.ProjectID.Get(); ok {
		data.ProjectID = types.StringValue(v)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationApiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationApiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	items, err := r.client.ListOrgApiKeys(ctx, neon.ListOrgApiKeysParams{
		OrgID: data.OrgID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list organization API keys", err.Error())
		return
	}

	for i := range items {
		if items[i].ID == data.ID.ValueInt64() {
			data.Name = types.StringValue(items[i].Name)
			data.CreatedAt = types.StringValue(items[i].CreatedAt.String())
			if v, ok := items[i].ProjectID.Get(); ok {
				data.ProjectID = types.StringValue(v)
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *organizationApiKeyResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Organization API keys do not support updates. All attributes require replacement.",
	)
}

func (r *organizationApiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationApiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeOrgApiKey(ctx, neon.RevokeOrgApiKeyParams{
		KeyID: data.ID.ValueInt64(),
		OrgID: data.OrgID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to revoke organization API key", err.Error())
		return
	}
}
