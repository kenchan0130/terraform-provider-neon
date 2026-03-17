package connection_uri

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type connectionURIModel struct {
	ProjectID    types.String `tfsdk:"project_id"`
	BranchID     types.String `tfsdk:"branch_id"`
	EndpointID   types.String `tfsdk:"endpoint_id"`
	DatabaseName types.String `tfsdk:"database_name"`
	RoleName     types.String `tfsdk:"role_name"`
	Pooled       types.Bool   `tfsdk:"pooled"`
	URI          types.String `tfsdk:"uri"`
}

func fetchConnectionURI(ctx context.Context, client *neon.Client, data *connectionURIModel) diag.Diagnostics {
	var diags diag.Diagnostics

	params := neon.GetConnectionURIParams{
		ProjectID:    data.ProjectID.ValueString(),
		DatabaseName: data.DatabaseName.ValueString(),
		RoleName:     data.RoleName.ValueString(),
	}

	if !data.BranchID.IsNull() {
		params.BranchID = neon.NewOptString(data.BranchID.ValueString())
	}

	if !data.EndpointID.IsNull() {
		params.EndpointID = neon.NewOptString(data.EndpointID.ValueString())
	}

	if !data.Pooled.IsNull() {
		params.Pooled = neon.NewOptBool(data.Pooled.ValueBool())
	}

	result, err := client.GetConnectionURI(ctx, params)
	if err != nil {
		diags.AddError("Failed to read connection URI", err.Error())
		return diags
	}

	data.URI = types.StringValue(result.URI)

	return diags
}
