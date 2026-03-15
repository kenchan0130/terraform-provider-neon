package role

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type roleModel struct {
	ProjectID            types.String `tfsdk:"project_id"`
	BranchID             types.String `tfsdk:"branch_id"`
	Name                 types.String `tfsdk:"name"`
	Protected            types.Bool   `tfsdk:"protected"`
	AuthenticationMethod types.String `tfsdk:"authentication_method"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

func fetchRole(ctx context.Context, client *neon.Client, data *roleModel) diag.Diagnostics {
	var diags diag.Diagnostics

	result, err := client.GetProjectBranchRole(ctx, neon.GetProjectBranchRoleParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
		RoleName:  data.Name.ValueString(),
	})
	if err != nil {
		diags.AddError("Failed to read role", err.Error())
		return diags
	}

	r := &result.Role
	data.BranchID = types.StringValue(r.BranchID)
	data.Name = types.StringValue(r.Name)

	if r.Protected.IsSet() {
		data.Protected = types.BoolValue(r.Protected.Value)
	} else {
		data.Protected = types.BoolNull()
	}

	if r.AuthenticationMethod.IsSet() {
		data.AuthenticationMethod = types.StringValue(r.AuthenticationMethod.Value)
	} else {
		data.AuthenticationMethod = types.StringNull()
	}

	data.CreatedAt = types.StringValue(r.CreatedAt.String())
	data.UpdatedAt = types.StringValue(r.UpdatedAt.String())

	return diags
}
