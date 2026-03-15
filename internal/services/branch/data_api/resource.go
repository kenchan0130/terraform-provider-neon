package data_api

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
	"github.com/kenchan0130/terraform-provider-neon/internal/neonerror"
)

var (
	_ resource.Resource                = &branchDataAPIResource{}
	_ resource.ResourceWithConfigure   = &branchDataAPIResource{}
	_ resource.ResourceWithImportState = &branchDataAPIResource{}
)

type branchDataAPIResource struct {
	client *neon.Client
}

type branchDataAPIResourceModel struct {
	ProjectID    types.String `tfsdk:"project_id"`
	BranchID     types.String `tfsdk:"branch_id"`
	DatabaseName types.String `tfsdk:"database_name"`
	URL          types.String `tfsdk:"url"`
	Status       types.String `tfsdk:"status"`
}

func NewResource() resource.Resource {
	return &branchDataAPIResource{}
}

func (r *branchDataAPIResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_data_api"
}

func (r *branchDataAPIResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Neon branch Data API.",
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
			"database_name": schema.StringAttribute{
				Description: "The database name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"url": schema.StringAttribute{
				Description: "The Data API URL.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The status of the Data API.",
				Computed:    true,
			},
		},
	}
}

func (r *branchDataAPIResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *branchDataAPIResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data branchDataAPIResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := neon.CreateProjectBranchDataAPIParams{
		ProjectID:    data.ProjectID.ValueString(),
		BranchID:     data.BranchID.ValueString(),
		DatabaseName: data.DatabaseName.ValueString(),
	}

	_, err := r.client.CreateProjectBranchDataAPI(ctx, neon.OptDataAPICreateRequest{}, params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create branch data API", err.Error())
		return
	}

	// Read to get full state since Create only returns URL.
	readResult, err := r.client.GetProjectBranchDataAPI(ctx, neon.GetProjectBranchDataAPIParams{
		ProjectID:    data.ProjectID.ValueString(),
		BranchID:     data.BranchID.ValueString(),
		DatabaseName: data.DatabaseName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read branch data API after create", err.Error())
		return
	}

	mapDataAPIResponseToModel(readResult, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *branchDataAPIResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data branchDataAPIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetProjectBranchDataAPI(ctx, neon.GetProjectBranchDataAPIParams{
		ProjectID:    data.ProjectID.ValueString(),
		BranchID:     data.BranchID.ValueString(),
		DatabaseName: data.DatabaseName.ValueString(),
	})
	if err != nil {
		if neonerror.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read branch data API", err.Error())
		return
	}

	mapDataAPIResponseToModel(result, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *branchDataAPIResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan branchDataAPIResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state branchDataAPIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UpdateProjectBranchDataAPI(ctx, neon.OptDataAPIUpdateRequest{}, neon.UpdateProjectBranchDataAPIParams{
		ProjectID:    state.ProjectID.ValueString(),
		BranchID:     state.BranchID.ValueString(),
		DatabaseName: state.DatabaseName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update branch data API", err.Error())
		return
	}

	// Read to get updated state.
	readResult, err := r.client.GetProjectBranchDataAPI(ctx, neon.GetProjectBranchDataAPIParams{
		ProjectID:    state.ProjectID.ValueString(),
		BranchID:     state.BranchID.ValueString(),
		DatabaseName: state.DatabaseName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read branch data API after update", err.Error())
		return
	}

	mapDataAPIResponseToModel(readResult, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *branchDataAPIResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data branchDataAPIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteProjectBranchDataAPI(ctx, neon.DeleteProjectBranchDataAPIParams{
		ProjectID:    data.ProjectID.ValueString(),
		BranchID:     data.BranchID.ValueString(),
		DatabaseName: data.DatabaseName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete branch data API", err.Error())
		return
	}
}

func (r *branchDataAPIResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: {project_id}/{branch_id}/{database_name}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("branch_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_name"), parts[2])...)
}

func mapDataAPIResponseToModel(r *neon.DataAPIReponse, data *branchDataAPIResourceModel) {
	data.URL = types.StringValue(formatURL(r.URL))
	data.Status = types.StringValue(r.Status)
}

func formatURL(u url.URL) string {
	return u.String()
}
