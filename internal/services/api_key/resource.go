package api_key

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
	_ resource.Resource              = &apiKeyResource{}
	_ resource.ResourceWithConfigure = &apiKeyResource{}
)

type apiKeyResource struct {
	client *neon.Client
}

type apiKeyResourceModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Key       types.String `tfsdk:"key"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewResource() resource.Resource {
	return &apiKeyResource{}
}

func (r *apiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *apiKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Neon API key. Import is not supported because the API key secret is only available at creation time.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The API key ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "A user-specified API key name.",
				Required:    true,
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

func (r *apiKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateApiKey(ctx, &neon.ApiKeyCreateRequest{
		KeyName: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create API key", err.Error())
		return
	}

	data.ID = types.Int64Value(result.ID)
	data.Name = types.StringValue(result.Name)
	data.Key = types.StringValue(result.Key)
	data.CreatedAt = types.StringValue(result.CreatedAt.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The Neon API does not provide a single-get endpoint for API keys,
	// so we list all keys and find the matching one.
	items, err := r.client.ListApiKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list API keys", err.Error())
		return
	}

	for i := range items {
		if items[i].ID == data.ID.ValueInt64() {
			data.Name = types.StringValue(items[i].Name)
			data.CreatedAt = types.StringValue(items[i].CreatedAt.String())
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *apiKeyResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"API keys do not support updates. All attributes require replacement.",
	)
}

func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeApiKey(ctx, neon.RevokeApiKeyParams{
		KeyID: data.ID.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to revoke API key", err.Error())
		return
	}
}
