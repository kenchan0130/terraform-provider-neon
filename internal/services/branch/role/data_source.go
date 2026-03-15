package role

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type roleDataSource struct {
	client *neon.Client
}

type roleDataSourceModel struct {
	ProjectID            types.String `tfsdk:"project_id"`
	BranchID             types.String `tfsdk:"branch_id"`
	Name                 types.String `tfsdk:"name"`
	Password             types.String `tfsdk:"password"`
	Protected            types.Bool   `tfsdk:"protected"`
	AuthenticationMethod types.String `tfsdk:"authentication_method"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

func NewDataSource() datasource.DataSource {
	return &roleDataSource{}
}

func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a Neon role.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The role name.",
				Required:    true,
			},
			"password": schema.StringAttribute{
				Description: "The role password.",
				Computed:    true,
				Sensitive:   true,
			},
			"protected": schema.BoolAttribute{
				Description: "Whether the role is protected.",
				Computed:    true,
			},
			"authentication_method": schema.StringAttribute{
				Description: "The authentication method used for the role.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp.",
				Computed:    true,
			},
		},
	}
}

func (d *roleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data roleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetProjectBranchRole(ctx, neon.GetProjectBranchRoleParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
		RoleName:  data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read role", err.Error())
		return
	}

	r := &result.Role
	data.BranchID = types.StringValue(r.BranchID)
	data.Name = types.StringValue(r.Name)

	if r.Password.IsSet() {
		data.Password = types.StringValue(r.Password.Value)
	} else {
		data.Password = types.StringNull()
	}

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

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
