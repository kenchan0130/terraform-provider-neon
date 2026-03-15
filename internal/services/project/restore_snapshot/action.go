package restore_snapshot

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ action.Action              = &restoreSnapshotAction{}
	_ action.ActionWithConfigure = &restoreSnapshotAction{}
)

type restoreSnapshotAction struct {
	client *neon.Client
}

type restoreSnapshotActionModel struct {
	ProjectID       string       `tfsdk:"project_id"`
	SnapshotID      string       `tfsdk:"snapshot_id"`
	Name            types.String `tfsdk:"name"`
	TargetBranchID  types.String `tfsdk:"target_branch_id"`
	FinalizeRestore types.Bool   `tfsdk:"finalize_restore"`
}

func NewAction() action.Action {
	return &restoreSnapshotAction{}
}

func (a *restoreSnapshotAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_snapshot"
}

func (a *restoreSnapshotAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Restores a snapshot in a Neon project.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"snapshot_id": schema.StringAttribute{
				Description: "The snapshot ID to restore.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "A name for the newly restored branch. If omitted, a default name will be generated.",
				Optional:    true,
			},
			"target_branch_id": schema.StringAttribute{
				Description: "The ID of the branch to restore the snapshot into. If not specified, the branch from which the snapshot was originally created will be used.",
				Optional:    true,
			},
			"finalize_restore": schema.BoolAttribute{
				Description: "Set to true to finalize the restore operation immediately. Defaults to false to allow previewing the restored snapshot data first.",
				Optional:    true,
			},
		},
	}
}

func (a *restoreSnapshotAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

	a.client = client
}

func (a *restoreSnapshotAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data restoreSnapshotActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var request neon.OptRestoreSnapshotReq

	hasBody := false
	var body neon.RestoreSnapshotReq

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		body.Name = neon.NewOptString(data.Name.ValueString())
		hasBody = true
	}

	if !data.TargetBranchID.IsNull() && !data.TargetBranchID.IsUnknown() {
		body.TargetBranchID = neon.NewOptString(data.TargetBranchID.ValueString())
		hasBody = true
	}

	if !data.FinalizeRestore.IsNull() && !data.FinalizeRestore.IsUnknown() {
		body.FinalizeRestore = neon.NewOptBool(data.FinalizeRestore.ValueBool())
		hasBody = true
	}

	if hasBody {
		request = neon.NewOptRestoreSnapshotReq(body)
	}

	_, err := a.client.RestoreSnapshot(ctx, request, neon.RestoreSnapshotParams{
		ProjectID:  data.ProjectID,
		SnapshotID: data.SnapshotID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to restore snapshot", err.Error())
	}
}
