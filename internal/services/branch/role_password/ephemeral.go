package role_password

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

var (
	_ ephemeral.EphemeralResource              = &rolePasswordEphemeral{}
	_ ephemeral.EphemeralResourceWithConfigure = &rolePasswordEphemeral{}
)

type rolePasswordEphemeral struct {
	client *neon.Client
}

type rolePasswordEphemeralModel struct {
	ProjectID types.String `tfsdk:"project_id"`
	BranchID  types.String `tfsdk:"branch_id"`
	RoleName  types.String `tfsdk:"role_name"`
	Password  types.String `tfsdk:"password"`
}

func NewEphemeralResource() ephemeral.EphemeralResource {
	return &rolePasswordEphemeral{}
}

func (e *rolePasswordEphemeral) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_password"
}

func (e *rolePasswordEphemeral) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the password for a Neon role. The password is ephemeral and will not be stored in Terraform state.",
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
			"password": schema.StringAttribute{
				Description: "The role password.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (e *rolePasswordEphemeral) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

	e.client = client
}

func (e *rolePasswordEphemeral) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data rolePasswordEphemeralModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := e.client.GetProjectBranchRolePassword(ctx, neon.GetProjectBranchRolePasswordParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
		RoleName:  data.RoleName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get role password", err.Error())
		return
	}

	switch res := result.(type) {
	case *neon.RolePasswordResponse:
		data.Password = types.StringValue(res.Password)
	case *neon.GetProjectBranchRolePasswordNotFound:
		resp.Diagnostics.AddError("Role not found", fmt.Sprintf("Message: %s", res.Message))
		return
	case *neon.GetProjectBranchRolePasswordPreconditionFailed:
		resp.Diagnostics.AddError("Password not available", fmt.Sprintf("Message: %s", res.Message))
		return
	default:
		resp.Diagnostics.AddError("Unexpected response type", fmt.Sprintf("Got: %T", result))
		return
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
