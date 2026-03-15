package organization

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type organizationDataSource struct {
	client *neon.Client
}

type organizationDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Handle             types.String `tfsdk:"handle"`
	Plan               types.String `tfsdk:"plan"`
	ManagedBy          types.String `tfsdk:"managed_by"`
	AllowHipaaProjects types.Bool   `tfsdk:"allow_hipaa_projects"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func NewDataSource() datasource.DataSource {
	return &organizationDataSource{}
}

func (d *organizationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *organizationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a Neon organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The organization ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The organization name.",
				Computed:    true,
			},
			"handle": schema.StringAttribute{
				Description: "The organization handle.",
				Computed:    true,
			},
			"plan": schema.StringAttribute{
				Description: "The organization billing plan.",
				Computed:    true,
			},
			"managed_by": schema.StringAttribute{
				Description: "How the organization is managed. Organizations created via the Console or the API are managed by `console`.",
				Computed:    true,
			},
			"allow_hipaa_projects": schema.BoolAttribute{
				Description: "If true, the organization is allowed to mark projects as HIPAA.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "A timestamp indicating when the organization was created, in RFC 3339 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "A timestamp indicating when the organization was updated, in RFC 3339 format.",
				Computed:    true,
			},
		},
	}
}

func (d *organizationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *organizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetOrganization(ctx, neon.GetOrganizationParams{
		OrgID: data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read organization", err.Error())
		return
	}

	data.ID = types.StringValue(result.ID)
	data.Name = types.StringValue(result.Name)
	data.Handle = types.StringValue(result.Handle)
	data.Plan = types.StringValue(result.Plan)
	data.ManagedBy = types.StringValue(result.ManagedBy)

	if result.AllowHipaaProjects.IsSet() {
		data.AllowHipaaProjects = types.BoolValue(result.AllowHipaaProjects.Value)
	} else {
		data.AllowHipaaProjects = types.BoolNull()
	}

	data.CreatedAt = types.StringValue(result.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(result.UpdatedAt.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
