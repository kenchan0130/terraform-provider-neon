package neon_auth_trusted_domain

import (
	"context"
	"fmt"
	"net/url"
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
	_ resource.Resource                = &neonAuthTrustedDomainResource{}
	_ resource.ResourceWithConfigure   = &neonAuthTrustedDomainResource{}
	_ resource.ResourceWithImportState = &neonAuthTrustedDomainResource{}
)

type neonAuthTrustedDomainResource struct {
	client *neon.Client
}

type neonAuthTrustedDomainResourceModel struct {
	ProjectID    types.String `tfsdk:"project_id"`
	BranchID     types.String `tfsdk:"branch_id"`
	Domain       types.String `tfsdk:"domain"`
	AuthProvider types.String `tfsdk:"auth_provider"`
}

func NewResource() resource.Resource {
	return &neonAuthTrustedDomainResource{}
}

func (r *neonAuthTrustedDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_neon_auth_trusted_domain"
}

func (r *neonAuthTrustedDomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a NeonAuth trusted domain on a branch.",
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
			"domain": schema.StringAttribute{
				Description: "The trusted domain URL.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"auth_provider": schema.StringAttribute{
				Description: "The authentication provider associated with this domain.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *neonAuthTrustedDomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *neonAuthTrustedDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data neonAuthTrustedDomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainURL, err := url.Parse(data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain URL", err.Error())
		return
	}

	// We need the auth_provider for the request. Read the NeonAuth integration to get it.
	integration, err := r.client.GetNeonAuth(ctx, neon.GetNeonAuthParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read NeonAuth integration", err.Error())
		return
	}

	createReq := &neon.NeonAuthAddDomainToRedirectURIWhitelistRequest{
		Domain:       *domainURL,
		AuthProvider: integration.AuthProvider,
	}

	err = r.client.AddBranchNeonAuthTrustedDomain(ctx, createReq, neon.AddBranchNeonAuthTrustedDomainParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to add NeonAuth trusted domain", err.Error())
		return
	}

	data.AuthProvider = types.StringValue(string(integration.AuthProvider))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *neonAuthTrustedDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data neonAuthTrustedDomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ListBranchNeonAuthTrustedDomains(ctx, neon.ListBranchNeonAuthTrustedDomainsParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list NeonAuth trusted domains", err.Error())
		return
	}

	for _, d := range result.Domains {
		if d.Domain == data.Domain.ValueString() {
			data.AuthProvider = types.StringValue(string(d.AuthProvider))
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *neonAuthTrustedDomainResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"NeonAuth trusted domain does not support updates. All attributes require replacement.",
	)
}

func (r *neonAuthTrustedDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data neonAuthTrustedDomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainURL, err := url.Parse(data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain URL", err.Error())
		return
	}

	deleteReq := &neon.NeonAuthDeleteDomainFromRedirectURIWhitelistRequest{
		AuthProvider: neon.NeonAuthSupportedAuthProvider(data.AuthProvider.ValueString()),
		Domains: []neon.NeonAuthDeleteDomainFromRedirectURIWhitelistItem{
			{Domain: *domainURL},
		},
	}

	err = r.client.DeleteBranchNeonAuthTrustedDomain(ctx, deleteReq, neon.DeleteBranchNeonAuthTrustedDomainParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete NeonAuth trusted domain", err.Error())
		return
	}
}

func (r *neonAuthTrustedDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: {project_id}/{branch_id}/{domain}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("branch_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[2])...)
}
