package set_default_branch

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ action.Action              = &setDefaultBranchAction{}
	_ action.ActionWithConfigure = &setDefaultBranchAction{}
)

type setDefaultBranchAction struct {
	client *neon.Client
}

type setDefaultBranchActionModel struct {
	ProjectID string `tfsdk:"project_id"`
	BranchID  string `tfsdk:"branch_id"`
}

func NewAction() action.Action {
	return &setDefaultBranchAction{}
}

func (a *setDefaultBranchAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_set_default_branch"
}

func (a *setDefaultBranchAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Sets a branch as the default branch for a Neon project.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID to set as the default.",
				Required:    true,
			},
		},
	}
}

func (a *setDefaultBranchAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *setDefaultBranchAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data setDefaultBranchActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := a.client.SetDefaultProjectBranch(ctx, neon.SetDefaultProjectBranchParams{
		ProjectID: data.ProjectID,
		BranchID:  data.BranchID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set default branch", err.Error())
	}
}
