package projects

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type projectsDataSource struct {
	client *neon.Client
}

type projectsDataSourceModel struct {
	Query    *queryModel    `tfsdk:"query"`
	Projects []projectModel `tfsdk:"projects"`
}

type queryModel struct {
	Search      types.String `tfsdk:"search"`
	OrgID       types.String `tfsdk:"org_id"`
	Recoverable types.Bool   `tfsdk:"recoverable"`
}

type projectModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	RegionID  types.String `tfsdk:"region_id"`
	PgVersion types.Int32  `tfsdk:"pg_version"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewDataSource() datasource.DataSource {
	return &projectsDataSource{}
}

func (d *projectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *projectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of Neon projects.",
		Attributes: map[string]schema.Attribute{
			"query": schema.SingleNestedAttribute{
				Description: "Query parameters for filtering projects.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"search": schema.StringAttribute{
						Description: "Search by project name or ID.",
						Optional:    true,
					},
					"org_id": schema.StringAttribute{
						Description: "Filter by organization ID.",
						Optional:    true,
					},
					"recoverable": schema.BoolAttribute{
						Description: "Show only deleted projects within the recovery window.",
						Optional:    true,
					},
				},
			},
			"projects": schema.ListNestedAttribute{
				Description: "The list of projects.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The project ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The project name.",
							Computed:    true,
						},
						"region_id": schema.StringAttribute{
							Description: "The region identifier.",
							Computed:    true,
						},
						"pg_version": schema.Int32Attribute{
							Description: "The Postgres version.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "A timestamp indicating when the project was created, in RFC 3339 format.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "A timestamp indicating when the project was last updated, in RFC 3339 format.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *projectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *projectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data projectsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var allProjects []neon.ProjectListItem
	var cursor neon.OptString

	for {
		params := neon.ListProjectsParams{
			Cursor: cursor,
			Limit:  neon.NewOptInt(400),
		}

		if data.Query != nil {
			if !data.Query.Search.IsNull() && !data.Query.Search.IsUnknown() {
				params.Search = neon.NewOptString(data.Query.Search.ValueString())
			}
			if !data.Query.OrgID.IsNull() && !data.Query.OrgID.IsUnknown() {
				params.OrgID = neon.NewOptString(data.Query.OrgID.ValueString())
			}
			if !data.Query.Recoverable.IsNull() && !data.Query.Recoverable.IsUnknown() {
				params.Recoverable = neon.NewOptBool(data.Query.Recoverable.ValueBool())
			}
		}

		result, err := d.client.ListProjects(ctx, params)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list projects", err.Error())
			return
		}

		allProjects = append(allProjects, result.Projects...)

		if result.Pagination.IsSet() && result.Pagination.Value.Cursor != "" {
			cursor = neon.NewOptString(result.Pagination.Value.Cursor)
		} else {
			break
		}
	}

	data.Projects = make([]projectModel, len(allProjects))
	for i, p := range allProjects {
		data.Projects[i] = projectModel{
			ID:        types.StringValue(p.ID),
			Name:      types.StringValue(p.Name),
			RegionID:  types.StringValue(p.RegionID),
			PgVersion: types.Int32Value(int32(p.PgVersion)),
			CreatedAt: types.StringValue(p.CreatedAt.Format(time.RFC3339)),
			UpdatedAt: types.StringValue(p.UpdatedAt.Format(time.RFC3339)),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
