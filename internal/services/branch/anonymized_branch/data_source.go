package anonymized_branch

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type anonymizedBranchDataSource struct {
	client *neon.Client
}

type anonymizedBranchDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	ProjectID    types.String `tfsdk:"project_id"`
	ParentID     types.String `tfsdk:"parent_id"`
	Name         types.String `tfsdk:"name"`
	MaskingRules types.List   `tfsdk:"masking_rules"`
	State        types.String `tfsdk:"state"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func NewDataSource() datasource.DataSource {
	return &anonymizedBranchDataSource{}
}

func (d *anonymizedBranchDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_anonymized_branch"
}

func (d *anonymizedBranchDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a Neon anonymized branch.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The branch ID.",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The project ID.",
				Required:    true,
			},
			"parent_id": schema.StringAttribute{
				Description: "The parent branch ID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The branch name.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "The current state of the anonymized branch (e.g., created, initialized, anonymizing, anonymized, error).",
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
		Blocks: map[string]schema.Block{
			"masking_rules": schema.ListNestedBlock{
				Description: "List of masking rules applied to the branch.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"database_name": schema.StringAttribute{
							Description: "The name of the database containing the table to be masked.",
							Computed:    true,
						},
						"schema_name": schema.StringAttribute{
							Description: "The name of the schema containing the table to be masked.",
							Computed:    true,
						},
						"table_name": schema.StringAttribute{
							Description: "The name of the table containing the column to be masked.",
							Computed:    true,
						},
						"column_name": schema.StringAttribute{
							Description: "The name of the column to be masked.",
							Computed:    true,
						},
						"masking_function": schema.StringAttribute{
							Description: "The PostgreSQL Anonymizer masking function applied.",
							Computed:    true,
						},
						"masking_value": schema.StringAttribute{
							Description: "A literal value set on the column when masking.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *anonymizedBranchDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *anonymizedBranchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data anonymizedBranchDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ProjectID.ValueString()
	branchID := data.ID.ValueString()

	branchResult, err := d.client.GetProjectBranch(ctx, neon.GetProjectBranchParams{
		ProjectID: projectID,
		BranchID:  branchID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read branch", err.Error())
		return
	}

	data.Name = types.StringValue(branchResult.Branch.Name)
	if v, ok := branchResult.Branch.ParentID.Get(); ok {
		data.ParentID = types.StringValue(v)
	} else {
		data.ParentID = types.StringNull()
	}

	maskingResult, err := d.client.GetMaskingRules(ctx, neon.GetMaskingRulesParams{
		ProjectID: projectID,
		BranchID:  branchID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read masking rules", err.Error())
		return
	}

	maskingRuleValues := make([]maskingRuleModel, len(maskingResult.MaskingRules))
	for i, rule := range maskingResult.MaskingRules {
		maskingRuleValues[i] = fromAPIMaskingRule(rule)
	}

	maskingRulesList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: maskingRuleAttrTypes()}, maskingRuleValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.MaskingRules = maskingRulesList

	statusResult, err := d.client.GetAnonymizedBranchStatus(ctx, neon.GetAnonymizedBranchStatusParams{
		ProjectID: projectID,
		BranchID:  branchID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read anonymized branch status", err.Error())
		return
	}

	data.State = types.StringValue(statusResult.State)
	data.CreatedAt = types.StringValue(statusResult.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	data.UpdatedAt = types.StringValue(statusResult.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
