package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonerror"
)

var (
	_ resource.Resource                = &databaseResource{}
	_ resource.ResourceWithConfigure   = &databaseResource{}
	_ resource.ResourceWithImportState = &databaseResource{}
)

type databaseResource struct {
	client *neon.Client
}

type databaseResourceModel struct {
	ID        types.Int64  `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	BranchID  types.String `tfsdk:"branch_id"`
	Name      types.String `tfsdk:"name"`
	OwnerName types.String `tfsdk:"owner_name"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewResource() resource.Resource {
	return &databaseResource{}
}

func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (r *databaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Neon database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The database ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "The project ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The database name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner_name": schema.StringAttribute{
				Description: "The name of the role that owns the database.",
				Required:    true,
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
			},
		},
	}
}

func (r *databaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *databaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data databaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &neon.DatabaseCreateRequest{
		Database: neon.DatabaseCreateRequestDatabase{
			Name:      data.Name.ValueString(),
			OwnerName: data.OwnerName.ValueString(),
		},
	}

	result, err := r.client.CreateProjectBranchDatabase(ctx, apiReq, neon.CreateProjectBranchDatabaseParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create database", err.Error())
		return
	}

	r.mapDatabaseToModel(&result.Database, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *databaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetProjectBranchDatabase(ctx, neon.GetProjectBranchDatabaseParams{
		ProjectID:    data.ProjectID.ValueString(),
		BranchID:     data.BranchID.ValueString(),
		DatabaseName: data.Name.ValueString(),
	})
	if err != nil {
		if neonerror.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read database", err.Error())
		return
	}

	r.mapDatabaseToModel(&result.Database, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *databaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data databaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &neon.DatabaseUpdateRequest{
		Database: neon.DatabaseUpdateRequestDatabase{
			OwnerName: neon.NewOptString(data.OwnerName.ValueString()),
		},
	}

	result, err := r.client.UpdateProjectBranchDatabase(ctx, apiReq, neon.UpdateProjectBranchDatabaseParams{
		ProjectID:    state.ProjectID.ValueString(),
		BranchID:     state.BranchID.ValueString(),
		DatabaseName: state.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update database", err.Error())
		return
	}

	r.mapDatabaseToModel(&result.Database, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *databaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectBranchDatabase(ctx, neon.DeleteProjectBranchDatabaseParams{
		ProjectID:    data.ProjectID.ValueString(),
		BranchID:     data.BranchID.ValueString(),
		DatabaseName: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete database", err.Error())
		return
	}
}

func (r *databaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[2])...)
}

func (r *databaseResource) mapDatabaseToModel(db *neon.Database, data *databaseResourceModel) {
	data.ID = types.Int64Value(db.ID)
	data.BranchID = types.StringValue(db.BranchID)
	data.Name = types.StringValue(db.Name)
	data.OwnerName = types.StringValue(db.OwnerName)
	data.CreatedAt = types.StringValue(db.CreatedAt.String())
	data.UpdatedAt = types.StringValue(db.UpdatedAt.String())
}
