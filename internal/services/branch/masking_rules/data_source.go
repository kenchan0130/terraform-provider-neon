package masking_rules

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type maskingRulesDataSource struct {
	client *neon.Client
}

type maskingRulesDataSourceModel struct {
	ProjectID    types.String `tfsdk:"project_id"`
	BranchID     types.String `tfsdk:"branch_id"`
	MaskingRules types.List   `tfsdk:"masking_rules"`
}

func NewDataSource() datasource.DataSource {
	return &maskingRulesDataSource{}
}

func (d *maskingRulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_masking_rules"
}

func (d *maskingRulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves masking rules for a Neon branch.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID.",
				Required:    true,
			},
			"masking_rules": schema.ListNestedAttribute{
				Description: "List of masking rules for the branch.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
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

func (d *maskingRulesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *maskingRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data maskingRulesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetMaskingRules(ctx, neon.GetMaskingRulesParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read masking rules", err.Error())
		return
	}

	maskingRuleValues := make([]maskingRuleModel, len(result.MaskingRules))
	for i, rule := range result.MaskingRules {
		maskingRuleValues[i] = fromAPIMaskingRule(rule)
	}

	maskingRulesList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: maskingRuleAttrTypes()}, maskingRuleValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.MaskingRules = maskingRulesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
