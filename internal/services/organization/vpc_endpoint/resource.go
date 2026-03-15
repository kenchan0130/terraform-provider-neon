package vpc_endpoint

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonerror"
)

var (
	_ resource.Resource                = &vpcEndpointResource{}
	_ resource.ResourceWithConfigure   = &vpcEndpointResource{}
	_ resource.ResourceWithImportState = &vpcEndpointResource{}
)

type vpcEndpointResource struct {
	client *neon.Client
}

type vpcEndpointResourceModel struct {
	OrgID                     types.String `tfsdk:"org_id"`
	RegionID                  types.String `tfsdk:"region_id"`
	VpcEndpointID             types.String `tfsdk:"vpc_endpoint_id"`
	Label                     types.String `tfsdk:"label"`
	State                     types.String `tfsdk:"state"`
	NumRestrictedProjects     types.Int64  `tfsdk:"num_restricted_projects"`
	ExampleRestrictedProjects types.List   `tfsdk:"example_restricted_projects"`
}

func NewResource() resource.Resource {
	return &vpcEndpointResource{}
}

func (r *vpcEndpointResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_vpc_endpoint"
}

func (r *vpcEndpointResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a VPC endpoint assignment for a Neon organization.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Description: "The Neon organization ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region_id": schema.StringAttribute{
				Description: "The Neon region ID. Azure regions are currently not supported.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vpc_endpoint_id": schema.StringAttribute{
				Description: "The VPC endpoint ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"label": schema.StringAttribute{
				Description: "A descriptive label for the VPC endpoint.",
				Required:    true,
			},
			"state": schema.StringAttribute{
				Description: "The current state of the VPC endpoint. Possible values are `new` or `accepted`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"num_restricted_projects": schema.Int64Attribute{
				Description: "The number of projects that are restricted to use this VPC endpoint.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"example_restricted_projects": schema.ListAttribute{
				Description: "A list of example projects that are restricted to use this VPC endpoint (at most 3).",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *vpcEndpointResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *vpcEndpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vpcEndpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AssignOrganizationVPCEndpoint(ctx, &neon.VPCEndpointAssignment{
		Label: data.Label.ValueString(),
	}, neon.AssignOrganizationVPCEndpointParams{
		OrgID:         data.OrgID.ValueString(),
		RegionID:      data.RegionID.ValueString(),
		VpcEndpointID: data.VpcEndpointID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to assign VPC endpoint", err.Error())
		return
	}

	r.readIntoModel(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vpcEndpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vpcEndpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetOrganizationVPCEndpointDetails(ctx, neon.GetOrganizationVPCEndpointDetailsParams{
		OrgID:         data.OrgID.ValueString(),
		RegionID:      data.RegionID.ValueString(),
		VpcEndpointID: data.VpcEndpointID.ValueString(),
	})
	if err != nil {
		if neonerror.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read VPC endpoint details", err.Error())
		return
	}

	mapVPCEndpointToModel(result, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vpcEndpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data vpcEndpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AssignOrganizationVPCEndpoint(ctx, &neon.VPCEndpointAssignment{
		Label: data.Label.ValueString(),
	}, neon.AssignOrganizationVPCEndpointParams{
		OrgID:         data.OrgID.ValueString(),
		RegionID:      data.RegionID.ValueString(),
		VpcEndpointID: data.VpcEndpointID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update VPC endpoint", err.Error())
		return
	}

	r.readIntoModel(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vpcEndpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vpcEndpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrganizationVPCEndpoint(ctx, neon.DeleteOrganizationVPCEndpointParams{
		OrgID:         data.OrgID.ValueString(),
		RegionID:      data.RegionID.ValueString(),
		VpcEndpointID: data.VpcEndpointID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete VPC endpoint", err.Error())
		return
	}
}

func (r *vpcEndpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: {org_id}/{region_id}/{vpc_endpoint_id}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vpc_endpoint_id"), parts[2])...)
}

func (r *vpcEndpointResource) readIntoModel(ctx context.Context, data *vpcEndpointResourceModel, diags *diag.Diagnostics) {
	result, err := r.client.GetOrganizationVPCEndpointDetails(ctx, neon.GetOrganizationVPCEndpointDetailsParams{
		OrgID:         data.OrgID.ValueString(),
		RegionID:      data.RegionID.ValueString(),
		VpcEndpointID: data.VpcEndpointID.ValueString(),
	})
	if err != nil {
		diags.AddError("Failed to read VPC endpoint details", err.Error())
		return
	}

	mapVPCEndpointToModel(result, data, diags)
}

func mapVPCEndpointToModel(ep *neon.VPCEndpointDetails, data *vpcEndpointResourceModel, diags *diag.Diagnostics) {
	data.VpcEndpointID = types.StringValue(ep.VpcEndpointID)
	data.Label = types.StringValue(ep.Label)
	data.State = types.StringValue(ep.State)
	data.NumRestrictedProjects = types.Int64Value(int64(ep.NumRestrictedProjects))

	elems := make([]attr.Value, len(ep.ExampleRestrictedProjects))
	for i, v := range ep.ExampleRestrictedProjects {
		elems[i] = types.StringValue(v)
	}
	list, d := types.ListValue(types.StringType, elems)
	diags.Append(d...)
	data.ExampleRestrictedProjects = list
}
