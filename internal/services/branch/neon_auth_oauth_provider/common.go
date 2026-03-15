package neon_auth_oauth_provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type oauthProviderModel struct {
	ID           types.String `tfsdk:"id"`
	ProjectID    types.String `tfsdk:"project_id"`
	BranchID     types.String `tfsdk:"branch_id"`
	Type         types.String `tfsdk:"type"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

func fetchOauthProvider(ctx context.Context, client *neon.Client, data *oauthProviderModel) diag.Diagnostics {
	var diags diag.Diagnostics

	result, err := client.ListBranchNeonAuthOauthProviders(ctx, neon.ListBranchNeonAuthOauthProvidersParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		diags.AddError("Failed to list NeonAuth OAuth providers", err.Error())
		return diags
	}

	for i := range result.Providers {
		if string(result.Providers[i].ID) == data.ID.ValueString() {
			p := &result.Providers[i]
			data.Type = types.StringValue(string(p.Type))

			if v, ok := p.ClientID.Get(); ok {
				data.ClientID = types.StringValue(v)
			} else {
				data.ClientID = types.StringNull()
			}

			if v, ok := p.ClientSecret.Get(); ok {
				data.ClientSecret = types.StringValue(v)
			} else {
				data.ClientSecret = types.StringNull()
			}

			return diags
		}
	}

	diags.AddError(
		"OAuth Provider Not Found",
		fmt.Sprintf("OAuth provider %q not found for branch %q in project %q.", data.ID.ValueString(), data.BranchID.ValueString(), data.ProjectID.ValueString()),
	)
	return diags
}
