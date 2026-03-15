package recover_project

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ action.Action              = &recoverProjectAction{}
	_ action.ActionWithConfigure = &recoverProjectAction{}
)

type recoverProjectAction struct {
	client *neon.Client
}

type recoverProjectActionModel struct {
	ProjectID string `tfsdk:"project_id"`
}

func NewAction() action.Action {
	return &recoverProjectAction{}
}

func (a *recoverProjectAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recover_project"
}

func (a *recoverProjectAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Recovers a deleted Neon project during the deletion grace period.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID to recover.",
				Required:    true,
			},
		},
	}
}

func (a *recoverProjectAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *recoverProjectAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data recoverProjectActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := a.client.RecoverProject(ctx, neon.RecoverProjectParams{
		ProjectID: data.ProjectID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to recover project", err.Error())
	}
}
