package branch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonerror"
)

var (
	_ resource.Resource                = &branchResource{}
	_ resource.ResourceWithConfigure   = &branchResource{}
	_ resource.ResourceWithImportState = &branchResource{}
)

type branchResource struct {
	client *neon.Client
}

type branchResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectID          types.String `tfsdk:"project_id"`
	Name               types.String `tfsdk:"name"`
	ParentID           types.String `tfsdk:"parent_id"`
	ParentLsn          types.String `tfsdk:"parent_lsn"`
	ParentTimestamp    types.String `tfsdk:"parent_timestamp"`
	Protected          types.Bool   `tfsdk:"protected"`
	Archived           types.Bool   `tfsdk:"archived"`
	InitSource         types.String `tfsdk:"init_source"`
	ExpiresAt          types.String `tfsdk:"expires_at"`
	CurrentState       types.String `tfsdk:"current_state"`
	LogicalSize        types.Int64  `tfsdk:"logical_size"`
	CreationSource     types.String `tfsdk:"creation_source"`
	Default            types.Bool   `tfsdk:"default"`
	ComputeTimeSeconds types.Int64  `tfsdk:"compute_time_seconds"`
	ActiveTimeSeconds  types.Int64  `tfsdk:"active_time_seconds"`
	WrittenDataBytes   types.Int64  `tfsdk:"written_data_bytes"`
	DataTransferBytes  types.Int64  `tfsdk:"data_transfer_bytes"`
	PendingState       types.String `tfsdk:"pending_state"`
	StateChangedAt     types.String `tfsdk:"state_changed_at"`
	LastResetAt        types.String `tfsdk:"last_reset_at"`
	RestoredFrom       types.String `tfsdk:"restored_from"`
	RestoredAs         types.String `tfsdk:"restored_as"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func NewResource() resource.Resource {
	return &branchResource{}
}

func (r *branchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch"
}

func (r *branchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Neon branch.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The branch ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "The project ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The branch name.",
				Optional:    true,
				Computed:    true,
			},
			"parent_id": schema.StringAttribute{
				Description: "The parent branch ID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parent_lsn": schema.StringAttribute{
				Description: "A Log Sequence Number (LSN) on the parent branch.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent_timestamp": schema.StringAttribute{
				Description: "A timestamp identifying a point in time on the parent branch (ISO 8601 format).",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"protected": schema.BoolAttribute{
				Description: "Whether the branch is protected.",
				Optional:    true,
				Computed:    true,
			},
			"archived": schema.BoolAttribute{
				Description: "Whether to create the branch as archived.",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"init_source": schema.StringAttribute{
				Description: "The source of initialization for the branch. Valid values are 'schema-only' and 'parent-data' (default).",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"expires_at": schema.StringAttribute{
				Description: "The timestamp when the branch is scheduled to expire and be automatically deleted (ISO 8601 / RFC 3339 format).",
				Optional:    true,
				Computed:    true,
			},
			"current_state": schema.StringAttribute{
				Description: "The current state of the branch.",
				Computed:    true,
			},
			"logical_size": schema.Int64Attribute{
				Description: "The logical size of the branch, in bytes.",
				Computed:    true,
			},
			"creation_source": schema.StringAttribute{
				Description: "The branch creation source.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default": schema.BoolAttribute{
				Description: "Whether the branch is the project's default branch.",
				Computed:    true,
			},
			"compute_time_seconds": schema.Int64Attribute{
				Description: "Compute time used by the branch, in seconds.",
				Computed:    true,
			},
			"active_time_seconds": schema.Int64Attribute{
				Description: "Active time for the branch, in seconds.",
				Computed:    true,
			},
			"written_data_bytes": schema.Int64Attribute{
				Description: "Written data for the branch, in bytes.",
				Computed:    true,
			},
			"data_transfer_bytes": schema.Int64Attribute{
				Description: "Data transfer for the branch, in bytes.",
				Computed:    true,
			},
			"pending_state": schema.StringAttribute{
				Description: "The pending state of the branch.",
				Computed:    true,
			},
			"state_changed_at": schema.StringAttribute{
				Description: "A timestamp indicating when the current state began.",
				Computed:    true,
			},
			"last_reset_at": schema.StringAttribute{
				Description: "A timestamp indicating when the branch was last reset.",
				Computed:    true,
			},
			"restored_from": schema.StringAttribute{
				Description: "The ID of the snapshot that was the restore source for this branch.",
				Computed:    true,
			},
			"restored_as": schema.StringAttribute{
				Description: "The ID of the target branch which was replaced when this branch was restored.",
				Computed:    true,
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

func (r *branchResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *branchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data branchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	branchReq := neon.CreateProjectBranchReqBranch{}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		branchReq.Name = neon.NewOptString(data.Name.ValueString())
	}
	if !data.ParentID.IsNull() && !data.ParentID.IsUnknown() {
		branchReq.ParentID = neon.NewOptString(data.ParentID.ValueString())
	}
	if !data.ParentLsn.IsNull() && !data.ParentLsn.IsUnknown() {
		branchReq.ParentLsn = neon.NewOptString(data.ParentLsn.ValueString())
	}
	if !data.ParentTimestamp.IsNull() && !data.ParentTimestamp.IsUnknown() {
		t, err := time.Parse(time.RFC3339, data.ParentTimestamp.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid parent_timestamp format", fmt.Sprintf("Expected RFC 3339 format: %s", err.Error()))
			return
		}
		branchReq.ParentTimestamp = neon.NewOptDateTime(t)
	}
	if !data.Protected.IsNull() && !data.Protected.IsUnknown() {
		branchReq.Protected = neon.NewOptBool(data.Protected.ValueBool())
	}
	if !data.Archived.IsNull() && !data.Archived.IsUnknown() {
		branchReq.Archived = neon.NewOptBool(data.Archived.ValueBool())
	}
	if !data.InitSource.IsNull() && !data.InitSource.IsUnknown() {
		branchReq.InitSource = neon.NewOptString(data.InitSource.ValueString())
	}
	if !data.ExpiresAt.IsNull() && !data.ExpiresAt.IsUnknown() {
		t, err := time.Parse(time.RFC3339, data.ExpiresAt.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid expires_at format", fmt.Sprintf("Expected RFC 3339 format: %s", err.Error()))
			return
		}
		branchReq.ExpiresAt = neon.NewOptDateTime(t)
	}

	apiReq := neon.NewOptCreateProjectBranchReq(neon.CreateProjectBranchReq{
		Branch: neon.NewOptCreateProjectBranchReqBranch(branchReq),
	})

	result, err := r.client.CreateProjectBranch(ctx, apiReq, neon.CreateProjectBranchParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create branch", err.Error())
		return
	}

	r.mapBranchToModel(&result.Branch, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *branchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data branchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetProjectBranch(ctx, neon.GetProjectBranchParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.ID.ValueString(),
	})
	if err != nil {
		if neonerror.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read branch", err.Error())
		return
	}

	r.mapBranchToModel(&result.Branch, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *branchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data branchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state branchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &neon.BranchUpdateRequest{
		Branch: neon.BranchUpdateRequestBranch{},
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		apiReq.Branch.Name = neon.NewOptString(data.Name.ValueString())
	}
	if !data.Protected.IsNull() && !data.Protected.IsUnknown() {
		apiReq.Branch.Protected = neon.NewOptBool(data.Protected.ValueBool())
	}
	if data.ExpiresAt.IsNull() {
		apiReq.Branch.ExpiresAt = neon.OptNilDateTime{Set: true, Null: true}
	} else if !data.ExpiresAt.IsUnknown() {
		t, err := time.Parse(time.RFC3339, data.ExpiresAt.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid expires_at format", fmt.Sprintf("Expected RFC 3339 format: %s", err.Error()))
			return
		}
		apiReq.Branch.ExpiresAt = neon.OptNilDateTime{Value: t, Set: true}
	}

	result, err := r.client.UpdateProjectBranch(ctx, apiReq, neon.UpdateProjectBranchParams{
		ProjectID: state.ProjectID.ValueString(),
		BranchID:  state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update branch", err.Error())
		return
	}

	r.mapBranchToModel(&result.Branch, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *branchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data branchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectBranch(ctx, neon.DeleteProjectBranchParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete branch", err.Error())
		return
	}
}

func (r *branchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: {project_id}/{branch_id}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *branchResource) mapBranchToModel(b *neon.Branch, data *branchResourceModel) {
	data.ID = types.StringValue(b.ID)
	data.ProjectID = types.StringValue(b.ProjectID)
	data.Name = types.StringValue(b.Name)

	if v, ok := b.ParentID.Get(); ok {
		data.ParentID = types.StringValue(v)
	} else {
		data.ParentID = types.StringNull()
	}

	if v, ok := b.ParentLsn.Get(); ok {
		data.ParentLsn = types.StringValue(v)
	} else {
		data.ParentLsn = types.StringNull()
	}

	if b.ParentTimestamp.IsSet() {
		data.ParentTimestamp = types.StringValue(b.ParentTimestamp.Value.Format(time.RFC3339))
	} else {
		data.ParentTimestamp = types.StringNull()
	}

	if v, ok := b.InitSource.Get(); ok {
		data.InitSource = types.StringValue(v)
	} else {
		data.InitSource = types.StringNull()
	}

	data.CurrentState = types.StringValue(string(b.CurrentState))

	if b.LogicalSize.IsSet() {
		data.LogicalSize = types.Int64Value(b.LogicalSize.Value)
	} else {
		data.LogicalSize = types.Int64Null()
	}

	data.CreationSource = types.StringValue(b.CreationSource)
	data.Default = types.BoolValue(b.Default)
	data.Protected = types.BoolValue(b.Protected)
	data.ComputeTimeSeconds = types.Int64Value(b.ComputeTimeSeconds)
	data.ActiveTimeSeconds = types.Int64Value(b.ActiveTimeSeconds)
	data.WrittenDataBytes = types.Int64Value(b.WrittenDataBytes)
	data.DataTransferBytes = types.Int64Value(b.DataTransferBytes)

	if v, ok := b.ExpiresAt.Get(); ok {
		data.ExpiresAt = types.StringValue(v.Format(time.RFC3339))
	} else {
		data.ExpiresAt = types.StringNull()
	}

	if b.PendingState.IsSet() {
		data.PendingState = types.StringValue(string(b.PendingState.Value))
	} else {
		data.PendingState = types.StringNull()
	}

	data.StateChangedAt = types.StringValue(b.StateChangedAt.Format(time.RFC3339))

	if b.LastResetAt.IsSet() {
		data.LastResetAt = types.StringValue(b.LastResetAt.Value.Format(time.RFC3339))
	} else {
		data.LastResetAt = types.StringNull()
	}

	if v, ok := b.RestoredFrom.Get(); ok {
		data.RestoredFrom = types.StringValue(v)
	} else {
		data.RestoredFrom = types.StringNull()
	}

	if v, ok := b.RestoredAs.Get(); ok {
		data.RestoredAs = types.StringValue(v)
	} else {
		data.RestoredAs = types.StringNull()
	}

	data.CreatedAt = types.StringValue(b.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(b.UpdatedAt.Format(time.RFC3339))
}
