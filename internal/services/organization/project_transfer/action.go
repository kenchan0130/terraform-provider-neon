package project_transfer

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ action.Action              = &projectTransferAction{}
	_ action.ActionWithConfigure = &projectTransferAction{}
)

type projectTransferAction struct {
	client *neon.Client
}

type projectTransferActionModel struct {
	SourceOrgID      types.String `tfsdk:"source_org_id"`
	DestinationOrgID types.String `tfsdk:"destination_org_id"`
	ProjectIDs       types.Set    `tfsdk:"project_ids"`
}

func NewAction() action.Action {
	return &projectTransferAction{}
}

func (a *projectTransferAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_project_transfer"
}

func (a *projectTransferAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Transfers projects between Neon organizations.",
		Attributes: map[string]schema.Attribute{
			"source_org_id": schema.StringAttribute{
				Description: "The source organization ID that currently owns the projects.",
				Required:    true,
			},
			"destination_org_id": schema.StringAttribute{
				Description: "The destination organization ID to transfer the projects to.",
				Required:    true,
			},
			"project_ids": schema.SetAttribute{
				Description: "The set of project IDs to transfer. Maximum of 400 project IDs.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (a *projectTransferAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *projectTransferAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data projectTransferActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var projectIDs []string
	resp.Diagnostics.Append(data.ProjectIDs.ElementsAs(ctx, &projectIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := a.client.TransferProjectsFromOrgToOrg(ctx, &neon.TransferProjectsToOrganizationRequest{
		DestinationOrgID: data.DestinationOrgID.ValueString(),
		ProjectIds:       projectIDs,
	}, neon.TransferProjectsFromOrgToOrgParams{
		SourceOrgID: data.SourceOrgID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to transfer projects", err.Error())
		return
	}

	switch res := result.(type) {
	case *neon.EmptyResponse:
		// Success
	case *neon.LimitsUnsatisfiedResponse:
		var details []string
		for _, l := range res.Limits {
			details = append(details, fmt.Sprintf("%s: expected %s, actual %s", l.Name, l.Expected, l.Actual))
		}
		resp.Diagnostics.AddError(
			"Transfer failed: destination organization limits unsatisfied",
			strings.Join(details, "; "),
		)
	case *neon.ProjectsWithIntegrationResponse:
		var details []string
		for _, p := range res.Projects {
			details = append(details, fmt.Sprintf("project %s has integration %s", p.ID, p.Integration))
		}
		resp.Diagnostics.AddError(
			"Transfer failed: projects have active integrations that must be removed first",
			strings.Join(details, "; "),
		)
	default:
		resp.Diagnostics.AddError("Unexpected response type", fmt.Sprintf("Got: %T", result))
	}
}
