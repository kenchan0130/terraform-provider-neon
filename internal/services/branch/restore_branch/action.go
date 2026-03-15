package restore_branch

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ action.Action              = &restoreBranchAction{}
	_ action.ActionWithConfigure = &restoreBranchAction{}
)

type restoreBranchAction struct {
	client *neon.Client
}

type restoreBranchActionModel struct {
	ProjectID        string       `tfsdk:"project_id"`
	BranchID         string       `tfsdk:"branch_id"`
	SourceBranchID   string       `tfsdk:"source_branch_id"`
	SourceLsn        types.String `tfsdk:"source_lsn"`
	SourceTimestamp  types.String `tfsdk:"source_timestamp"`
	PreserveUnderName types.String `tfsdk:"preserve_under_name"`
}

func NewAction() action.Action {
	return &restoreBranchAction{}
}

func (a *restoreBranchAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_branch"
}

func (a *restoreBranchAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Restores a branch to a previous state using a source branch, LSN, or timestamp.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The target branch ID to restore.",
				Required:    true,
			},
			"source_branch_id": schema.StringAttribute{
				Description: "The branch ID of the restore source branch.",
				Required:    true,
			},
			"source_lsn": schema.StringAttribute{
				Description: "A Log Sequence Number (LSN) on the source branch. The branch will be restored with data from this LSN.",
				Optional:    true,
			},
			"source_timestamp": schema.StringAttribute{
				Description: "A timestamp identifying a point in time on the source branch. Must be provided in ISO 8601 format.",
				Optional:    true,
			},
			"preserve_under_name": schema.StringAttribute{
				Description: "If not empty, the previous state of the branch will be saved to a branch with this name.",
				Optional:    true,
			},
		},
	}
}

func (a *restoreBranchAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *restoreBranchAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data restoreBranchActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := &neon.BranchRestoreRequest{
		SourceBranchID: data.SourceBranchID,
	}

	if !data.SourceLsn.IsNull() && !data.SourceLsn.IsUnknown() {
		request.SourceLsn = neon.NewOptString(data.SourceLsn.ValueString())
	}

	if !data.SourceTimestamp.IsNull() && !data.SourceTimestamp.IsUnknown() {
		t, err := time.Parse(time.RFC3339, data.SourceTimestamp.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid source_timestamp", fmt.Sprintf("Failed to parse timestamp: %s", err.Error()))
			return
		}
		request.SourceTimestamp = neon.NewOptDateTime(t)
	}

	if !data.PreserveUnderName.IsNull() && !data.PreserveUnderName.IsUnknown() {
		request.PreserveUnderName = neon.NewOptString(data.PreserveUnderName.ValueString())
	}

	_, err := a.client.RestoreProjectBranch(ctx, request, neon.RestoreProjectBranchParams{
		ProjectID: data.ProjectID,
		BranchID:  data.BranchID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to restore branch", err.Error())
	}
}
