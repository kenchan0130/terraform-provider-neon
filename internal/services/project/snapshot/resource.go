package snapshot

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
	_ resource.Resource                = &snapshotResource{}
	_ resource.ResourceWithConfigure   = &snapshotResource{}
	_ resource.ResourceWithImportState = &snapshotResource{}
)

type snapshotResource struct {
	client *neon.Client
}

type snapshotResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectID      types.String `tfsdk:"project_id"`
	BranchID       types.String `tfsdk:"branch_id"`
	Name           types.String `tfsdk:"name"`
	Lsn            types.String `tfsdk:"lsn"`
	Timestamp      types.String `tfsdk:"timestamp"`
	SourceBranchID types.String `tfsdk:"source_branch_id"`
	ExpiresAt      types.String `tfsdk:"expires_at"`
	Manual         types.Bool   `tfsdk:"manual"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

func NewResource() resource.Resource {
	return &snapshotResource{}
}

func (r *snapshotResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot"
}

func (r *snapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Neon snapshot.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The snapshot ID.",
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
			"branch_id": schema.StringAttribute{
				Description: "The branch ID to create the snapshot from.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "A name for the snapshot.",
				Optional:    true,
				Computed:    true,
			},
			"lsn": schema.StringAttribute{
				Description: "The target Log Sequence Number (LSN) to take the snapshot from. Must fall within the restore window. Cannot be used with `timestamp`.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"timestamp": schema.StringAttribute{
				Description: "The target timestamp for the snapshot. Must fall within the restore window. Use ISO 8601 format. Cannot be used with `lsn`.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_branch_id": schema.StringAttribute{
				Description: "The ID of the branch the snapshot was created from.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expires_at": schema.StringAttribute{
				Description: "The time at which the snapshot will be automatically deleted. Use ISO 8601 format.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"manual": schema.BoolAttribute{
				Description: "Whether the snapshot was manually created.",
				Computed:    true,
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

func (r *snapshotResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *snapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data snapshotResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := neon.CreateSnapshotParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		params.Name = neon.NewOptString(data.Name.ValueString())
	}
	if !data.Lsn.IsNull() && !data.Lsn.IsUnknown() {
		params.Lsn = neon.NewOptString(data.Lsn.ValueString())
	}
	if !data.Timestamp.IsNull() && !data.Timestamp.IsUnknown() {
		params.Timestamp = neon.NewOptString(data.Timestamp.ValueString())
	}
	if !data.ExpiresAt.IsNull() && !data.ExpiresAt.IsUnknown() {
		params.ExpiresAt = neon.NewOptString(data.ExpiresAt.ValueString())
	}

	result, err := r.client.CreateSnapshot(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create snapshot", err.Error())
		return
	}

	mapSnapshotToModel(&result.Snapshot, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *snapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data snapshotResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ListSnapshots(ctx, neon.ListSnapshotsParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list snapshots", err.Error())
		return
	}

	for i := range result.Snapshots {
		if result.Snapshots[i].ID == data.ID.ValueString() {
			mapSnapshotToModel(&result.Snapshots[i], &data)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *snapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan snapshotResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state snapshotResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &neon.SnapshotUpdateRequest{}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		updateReq.Snapshot.Name = neon.NewOptString(plan.Name.ValueString())
	}

	result, err := r.client.UpdateSnapshot(ctx, updateReq, neon.UpdateSnapshotParams{
		ProjectID:  state.ProjectID.ValueString(),
		SnapshotID: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update snapshot", err.Error())
		return
	}

	mapSnapshotToModel(&result.Snapshot, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *snapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data snapshotResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteSnapshot(ctx, neon.DeleteSnapshotParams{
		ProjectID:  data.ProjectID.ValueString(),
		SnapshotID: data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete snapshot", err.Error())
		return
	}
}

func (r *snapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: {project_id}/{snapshot_id}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func mapSnapshotToModel(s *neon.Snapshot, data *snapshotResourceModel) {
	data.ID = types.StringValue(s.ID)
	data.Name = types.StringValue(s.Name)
	data.CreatedAt = types.StringValue(s.CreatedAt)

	if v, ok := s.Lsn.Get(); ok {
		data.Lsn = types.StringValue(v)
	} else {
		data.Lsn = types.StringNull()
	}

	if v, ok := s.Timestamp.Get(); ok {
		data.Timestamp = types.StringValue(v)
	} else {
		data.Timestamp = types.StringNull()
	}

	if v, ok := s.SourceBranchID.Get(); ok {
		data.SourceBranchID = types.StringValue(v)
		if data.BranchID.IsNull() || data.BranchID.IsUnknown() {
			data.BranchID = types.StringValue(v)
		}
	} else {
		data.SourceBranchID = types.StringNull()
	}

	if v, ok := s.ExpiresAt.Get(); ok {
		data.ExpiresAt = types.StringValue(v)
	} else {
		data.ExpiresAt = types.StringNull()
	}

	if v, ok := s.Manual.Get(); ok {
		data.Manual = types.BoolValue(v)
	} else {
		data.Manual = types.BoolNull()
	}
}
