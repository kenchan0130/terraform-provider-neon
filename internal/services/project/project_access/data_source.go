package project_access

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type projectAccessDataSource struct {
	client *neon.Client
}

type projectAccessDataSourceModel struct {
	ProjectID      types.String `tfsdk:"project_id"`
	PermissionID   types.String `tfsdk:"permission_id"`
	GrantedToEmail types.String `tfsdk:"granted_to_email"`
	GrantedAt      types.String `tfsdk:"granted_at"`
}

func NewDataSource() datasource.DataSource {
	return &projectAccessDataSource{}
}

func (d *projectAccessDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_access"
}

func (d *projectAccessDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a project access permission.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"permission_id": schema.StringAttribute{
				Description: "The permission ID.",
				Required:    true,
			},
			"granted_to_email": schema.StringAttribute{
				Description: "The email address the project access was granted to.",
				Computed:    true,
			},
			"granted_at": schema.StringAttribute{
				Description: "The timestamp when the permission was granted.",
				Computed:    true,
			},
		},
	}
}

func (d *projectAccessDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}

func (d *projectAccessDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data projectAccessDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.ListProjectPermissions(ctx, neon.ListProjectPermissionsParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list project permissions", err.Error())
		return
	}

	for _, perm := range result.ProjectPermissions {
		if perm.ID == data.PermissionID.ValueString() {
			if perm.RevokedAt.IsSet() {
				resp.Diagnostics.AddError(
					"Permission Revoked",
					fmt.Sprintf("Permission %q has been revoked.", data.PermissionID.ValueString()),
				)
				return
			}
			data.GrantedToEmail = types.StringValue(perm.GrantedToEmail)
			data.GrantedAt = types.StringValue(perm.GrantedAt.String())
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"Permission Not Found",
		fmt.Sprintf("Permission %q not found in project %q.", data.PermissionID.ValueString(), data.ProjectID.ValueString()),
	)
}
