package role_password

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type rolePasswordModel struct {
	ProjectID types.String `tfsdk:"project_id"`
	BranchID  types.String `tfsdk:"branch_id"`
	RoleName  types.String `tfsdk:"role_name"`
	Password  types.String `tfsdk:"password"`
}

func fetchRolePassword(ctx context.Context, client *neon.Client, data *rolePasswordModel) diag.Diagnostics {
	var diags diag.Diagnostics

	result, err := client.GetProjectBranchRolePassword(ctx, neon.GetProjectBranchRolePasswordParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
		RoleName:  data.RoleName.ValueString(),
	})
	if err != nil {
		diags.AddError("Failed to get role password", err.Error())
		return diags
	}

	switch res := result.(type) {
	case *neon.RolePasswordResponse:
		data.Password = types.StringValue(res.Password)
	case *neon.GetProjectBranchRolePasswordNotFound:
		diags.AddError("Role not found", fmt.Sprintf("Message: %s", res.Message))
	case *neon.GetProjectBranchRolePasswordPreconditionFailed:
		diags.AddError("Password not available", fmt.Sprintf("Message: %s", res.Message))
	default:
		diags.AddError("Unexpected response type", fmt.Sprintf("Got: %T", result))
	}

	return diags
}
