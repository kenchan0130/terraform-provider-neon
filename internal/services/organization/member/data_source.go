package member

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type organizationMembersDataSource struct {
	client *neon.Client
}

type organizationMembersDataSourceModel struct {
	OrgID   types.String   `tfsdk:"org_id"`
	Query   *membersQuery  `tfsdk:"query"`
	Members []memberModel  `tfsdk:"members"`
}

type membersQuery struct {
	SortBy    types.String `tfsdk:"sort_by"`
	SortOrder types.String `tfsdk:"sort_order"`
}

type memberModel struct {
	ID       types.String `tfsdk:"id"`
	UserID   types.String `tfsdk:"user_id"`
	Role     types.String `tfsdk:"role"`
	Email    types.String `tfsdk:"email"`
	HasMfa   types.Bool   `tfsdk:"has_mfa"`
	JoinedAt types.String `tfsdk:"joined_at"`
}

func NewDataSource() datasource.DataSource {
	return &organizationMembersDataSource{}
}

func (d *organizationMembersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_members"
}

func (d *organizationMembersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the list of members for a Neon organization.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Description: "The Neon organization ID.",
				Required:    true,
			},
			"query": schema.SingleNestedAttribute{
				Description: "Query parameters for filtering and sorting members.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"sort_by": schema.StringAttribute{
						Description: "Sort members by the specified field. Possible values: `email`, `role`, `joined_at`. Defaults to `joined_at`.",
						Optional:    true,
					},
					"sort_order": schema.StringAttribute{
						Description: "Sort order. Possible values: `asc`, `desc`.",
						Optional:    true,
					},
					},
			},
			"members": schema.ListNestedAttribute{
				Description: "The list of organization members.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The member ID.",
							Computed:    true,
						},
						"user_id": schema.StringAttribute{
							Description: "The user ID.",
							Computed:    true,
						},
						"role": schema.StringAttribute{
							Description: "The member role. Possible values are `admin` or `member`.",
							Computed:    true,
						},
						"email": schema.StringAttribute{
							Description: "The member email address.",
							Computed:    true,
						},
						"has_mfa": schema.BoolAttribute{
							Description: "Whether the member has MFA (TOTP) enabled.",
							Computed:    true,
						},
						"joined_at": schema.StringAttribute{
							Description: "A timestamp indicating when the member joined the organization, in RFC 3339 format.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *organizationMembersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *organizationMembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationMembersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var allMembers []neon.MemberWithUser
	var cursor neon.OptString

	params := neon.GetOrganizationMembersParams{
		OrgID: data.OrgID.ValueString(),
	}

	if data.Query != nil {
		if !data.Query.SortBy.IsNull() && !data.Query.SortBy.IsUnknown() {
			params.SortBy = neon.NewOptGetOrganizationMembersSortBy(neon.GetOrganizationMembersSortBy(data.Query.SortBy.ValueString()))
		}
		if !data.Query.SortOrder.IsNull() && !data.Query.SortOrder.IsUnknown() {
			params.SortOrder = neon.NewOptSortOrderParam(neon.SortOrderParam(data.Query.SortOrder.ValueString()))
		}
	}

	for {
		params.Cursor = cursor

		result, err := d.client.GetOrganizationMembers(ctx, params)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read organization members", err.Error())
			return
		}

		allMembers = append(allMembers, result.Members...)

		if result.Pagination.IsSet() && result.Pagination.Value.Next.IsSet() && result.Pagination.Value.Next.Value != "" {
			cursor = result.Pagination.Value.Next
		} else {
			break
		}
	}

	data.Members = make([]memberModel, len(allMembers))
	for i, m := range allMembers {
		member := memberModel{
			ID:     types.StringValue(m.Member.ID.String()),
			UserID: types.StringValue(m.Member.UserID.String()),
			Role:   types.StringValue(string(m.Member.Role)),
			Email:  types.StringValue(m.User.Email),
		}

		if m.User.HasMfa.IsSet() {
			member.HasMfa = types.BoolValue(m.User.HasMfa.Value)
		} else {
			member.HasMfa = types.BoolNull()
		}

		if m.Member.JoinedAt.IsSet() {
			member.JoinedAt = types.StringValue(m.Member.JoinedAt.Value.Format(time.RFC3339))
		} else {
			member.JoinedAt = types.StringNull()
		}

		data.Members[i] = member
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
