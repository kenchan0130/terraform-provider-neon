package branches

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type branchesDataSource struct {
	client *neon.Client
}

type branchesDataSourceModel struct {
	ProjectID types.String  `tfsdk:"project_id"`
	Query     *queryModel   `tfsdk:"query"`
	Branches  []branchModel `tfsdk:"branches"`
}

type queryModel struct {
	Search    types.String `tfsdk:"search"`
	SortBy    types.String `tfsdk:"sort_by"`
	SortOrder types.String `tfsdk:"sort_order"`
}

type branchModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	ParentID     types.String `tfsdk:"parent_id"`
	CurrentState types.String `tfsdk:"current_state"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func NewDataSource() datasource.DataSource {
	return &branchesDataSource{}
}

func (d *branchesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branches"
}

func (d *branchesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the list of branches for a Neon project.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"query": schema.SingleNestedAttribute{
				Description: "Query parameters for filtering and sorting branches.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"search": schema.StringAttribute{
						Description: "Search by branch name or ID.",
						Optional:    true,
					},
					"sort_by": schema.StringAttribute{
						Description: "Sort field. Possible values are `name`, `created_at`, `updated_at`.",
						Optional:    true,
					},
					"sort_order": schema.StringAttribute{
						Description: "Sort order. Possible values are `asc`, `desc`.",
						Optional:    true,
					},
				},
			},
			"branches": schema.ListNestedAttribute{
				Description: "The list of branches.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The branch ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The branch name.",
							Computed:    true,
						},
						"parent_id": schema.StringAttribute{
							Description: "The parent branch ID.",
							Computed:    true,
						},
						"current_state": schema.StringAttribute{
							Description: "The current state of the branch.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "A timestamp indicating when the branch was created, in RFC 3339 format.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "A timestamp indicating when the branch was last updated, in RFC 3339 format.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *branchesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *branchesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data branchesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var allBranches []neon.Branch
	var cursor neon.OptString

	params := neon.ListProjectBranchesParams{
		ProjectID: data.ProjectID.ValueString(),
	}

	if data.Query != nil {
		if !data.Query.Search.IsNull() && !data.Query.Search.IsUnknown() {
			params.Search = neon.NewOptString(data.Query.Search.ValueString())
		}
		if !data.Query.SortBy.IsNull() && !data.Query.SortBy.IsUnknown() {
			params.SortBy = neon.NewOptListProjectBranchesSortBy(neon.ListProjectBranchesSortBy(data.Query.SortBy.ValueString()))
		}
		if !data.Query.SortOrder.IsNull() && !data.Query.SortOrder.IsUnknown() {
			params.SortOrder = neon.NewOptSortOrderParam(neon.SortOrderParam(data.Query.SortOrder.ValueString()))
		}
	}

	for {
		params.Cursor = cursor

		result, err := d.client.ListProjectBranches(ctx, params)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list branches", err.Error())
			return
		}

		allBranches = append(allBranches, result.Branches...)

		if result.Pagination.IsSet() && result.Pagination.Value.Next.IsSet() && result.Pagination.Value.Next.Value != "" {
			cursor = result.Pagination.Value.Next
		} else {
			break
		}
	}

	data.Branches = make([]branchModel, len(allBranches))
	for i, b := range allBranches {
		branch := branchModel{
			ID:           types.StringValue(b.ID),
			Name:         types.StringValue(b.Name),
			CurrentState: types.StringValue(string(b.CurrentState)),
			CreatedAt:    types.StringValue(b.CreatedAt.Format(time.RFC3339)),
			UpdatedAt:    types.StringValue(b.UpdatedAt.Format(time.RFC3339)),
		}

		if b.ParentID.IsSet() {
			branch.ParentID = types.StringValue(b.ParentID.Value)
		} else {
			branch.ParentID = types.StringNull()
		}

		data.Branches[i] = branch
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
