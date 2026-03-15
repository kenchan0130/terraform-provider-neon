package role_password_reset

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ action.Action              = &rolePasswordResetAction{}
	_ action.ActionWithConfigure = &rolePasswordResetAction{}
)

type rolePasswordResetAction struct {
	client *neon.Client
}

type rolePasswordResetActionModel struct {
	ProjectID string `tfsdk:"project_id"`
	BranchID  string `tfsdk:"branch_id"`
	RoleName  string `tfsdk:"role_name"`
}

func NewAction() action.Action {
	return &rolePasswordResetAction{}
}

func (a *rolePasswordResetAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_password_reset"
}

func (a *rolePasswordResetAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resets the password for a Neon role.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID.",
				Required:    true,
			},
			"role_name": schema.StringAttribute{
				Description: "The role name.",
				Required:    true,
			},
		},
	}
}

func (a *rolePasswordResetAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *rolePasswordResetAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data rolePasswordResetActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := a.client.ResetProjectBranchRolePassword(ctx, neon.ResetProjectBranchRolePasswordParams{
		ProjectID: data.ProjectID,
		BranchID:  data.BranchID,
		RoleName:  data.RoleName,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset role password", err.Error())
	}
}
