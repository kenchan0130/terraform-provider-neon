package branch

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type branchDataSource struct {
	client *neon.Client
}

type branchDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectID          types.String `tfsdk:"project_id"`
	Name               types.String `tfsdk:"name"`
	ParentID           types.String `tfsdk:"parent_id"`
	ParentLsn          types.String `tfsdk:"parent_lsn"`
	ParentTimestamp    types.String `tfsdk:"parent_timestamp"`
	Protected          types.Bool   `tfsdk:"protected"`
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

func NewDataSource() datasource.DataSource {
	return &branchDataSource{}
}

func (d *branchDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch"
}

func (d *branchDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a Neon branch.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The branch ID.",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The project ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The branch name.",
				Computed:    true,
			},
			"parent_id": schema.StringAttribute{
				Description: "The parent branch ID.",
				Computed:    true,
			},
			"parent_lsn": schema.StringAttribute{
				Description: "A Log Sequence Number (LSN) on the parent branch.",
				Computed:    true,
			},
			"parent_timestamp": schema.StringAttribute{
				Description: "A timestamp identifying a point in time on the parent branch (ISO 8601 format).",
				Computed:    true,
			},
			"protected": schema.BoolAttribute{
				Description: "Whether the branch is protected.",
				Computed:    true,
			},
			"init_source": schema.StringAttribute{
				Description: "The source of initialization for the branch.",
				Computed:    true,
			},
			"expires_at": schema.StringAttribute{
				Description: "The timestamp when the branch is scheduled to expire and be automatically deleted (ISO 8601 / RFC 3339 format).",
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
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp.",
				Computed:    true,
			},
		},
	}
}

func (d *branchDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}

func (d *branchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data branchDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetProjectBranch(ctx, neon.GetProjectBranchParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read branch", err.Error())
		return
	}

	b := &result.Branch
	var rm branchResourceModel
	r := &branchResource{}
	r.mapBranchToModel(b, &rm)

	data.ID = rm.ID
	data.ProjectID = rm.ProjectID
	data.Name = rm.Name
	data.ParentID = rm.ParentID
	data.ParentLsn = rm.ParentLsn
	data.ParentTimestamp = rm.ParentTimestamp
	data.Protected = rm.Protected
	data.InitSource = rm.InitSource
	data.ExpiresAt = rm.ExpiresAt
	data.CurrentState = rm.CurrentState
	data.LogicalSize = rm.LogicalSize
	data.CreationSource = rm.CreationSource
	data.Default = rm.Default
	data.ComputeTimeSeconds = rm.ComputeTimeSeconds
	data.ActiveTimeSeconds = rm.ActiveTimeSeconds
	data.WrittenDataBytes = rm.WrittenDataBytes
	data.DataTransferBytes = rm.DataTransferBytes
	data.PendingState = rm.PendingState
	data.StateChangedAt = rm.StateChangedAt
	data.LastResetAt = rm.LastResetAt
	data.RestoredFrom = rm.RestoredFrom
	data.RestoredAs = rm.RestoredAs
	data.CreatedAt = rm.CreatedAt
	data.UpdatedAt = rm.UpdatedAt

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
