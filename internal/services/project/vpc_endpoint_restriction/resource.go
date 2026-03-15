package vpc_endpoint_restriction

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
	_ resource.Resource                = &vpcEndpointRestrictionResource{}
	_ resource.ResourceWithConfigure   = &vpcEndpointRestrictionResource{}
	_ resource.ResourceWithImportState = &vpcEndpointRestrictionResource{}
)

type vpcEndpointRestrictionResource struct {
	client *neon.Client
}

type vpcEndpointRestrictionResourceModel struct {
	ProjectID     types.String `tfsdk:"project_id"`
	VpcEndpointID types.String `tfsdk:"vpc_endpoint_id"`
	Label         types.String `tfsdk:"label"`
}

func NewResource() resource.Resource {
	return &vpcEndpointRestrictionResource{}
}

func (r *vpcEndpointRestrictionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_vpc_endpoint_restriction"
}

func (r *vpcEndpointRestrictionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a VPC endpoint restriction for a Neon project. When set, the project only accepts connections from the specified VPC endpoint.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
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
		},
	}
}

func (r *vpcEndpointRestrictionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *vpcEndpointRestrictionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vpcEndpointRestrictionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AssignProjectVPCEndpoint(ctx, &neon.VPCEndpointAssignment{
		Label: data.Label.ValueString(),
	}, neon.AssignProjectVPCEndpointParams{
		ProjectID:     data.ProjectID.ValueString(),
		VpcEndpointID: data.VpcEndpointID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to assign VPC endpoint restriction", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vpcEndpointRestrictionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vpcEndpointRestrictionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ListProjectVPCEndpoints(ctx, neon.ListProjectVPCEndpointsParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list VPC endpoint restrictions", err.Error())
		return
	}

	for _, ep := range result.Endpoints {
		if ep.VpcEndpointID == data.VpcEndpointID.ValueString() {
			data.Label = types.StringValue(ep.Label)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *vpcEndpointRestrictionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vpcEndpointRestrictionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AssignProjectVPCEndpoint(ctx, &neon.VPCEndpointAssignment{
		Label: plan.Label.ValueString(),
	}, neon.AssignProjectVPCEndpointParams{
		ProjectID:     plan.ProjectID.ValueString(),
		VpcEndpointID: plan.VpcEndpointID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update VPC endpoint restriction", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vpcEndpointRestrictionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vpcEndpointRestrictionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteProjectVPCEndpoint(ctx, neon.DeleteProjectVPCEndpointParams{
		ProjectID:     data.ProjectID.ValueString(),
		VpcEndpointID: data.VpcEndpointID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete VPC endpoint restriction", err.Error())
		return
	}
}

func (r *vpcEndpointRestrictionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: {project_id}/{vpc_endpoint_id}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vpc_endpoint_id"), parts[1])...)
}
