package masking_rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonerror"
)

var (
	_ resource.Resource                = &maskingRulesResource{}
	_ resource.ResourceWithConfigure   = &maskingRulesResource{}
	_ resource.ResourceWithImportState = &maskingRulesResource{}
)

type maskingRulesResource struct {
	client *neon.Client
}

type maskingRulesResourceModel struct {
	ProjectID    types.String `tfsdk:"project_id"`
	BranchID     types.String `tfsdk:"branch_id"`
	MaskingRules types.List   `tfsdk:"masking_rules"`
}

type maskingRuleModel struct {
	DatabaseName    types.String `tfsdk:"database_name"`
	SchemaName      types.String `tfsdk:"schema_name"`
	TableName       types.String `tfsdk:"table_name"`
	ColumnName      types.String `tfsdk:"column_name"`
	MaskingFunction types.String `tfsdk:"masking_function"`
	MaskingValue    types.String `tfsdk:"masking_value"`
}

func maskingRuleAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"database_name":    types.StringType,
		"schema_name":      types.StringType,
		"table_name":       types.StringType,
		"column_name":      types.StringType,
		"masking_function": types.StringType,
		"masking_value":    types.StringType,
	}
}

func NewResource() resource.Resource {
	return &maskingRulesResource{}
}

func (r *maskingRulesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_masking_rules"
}

func (r *maskingRulesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages masking rules for a Neon branch.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"masking_rules": schema.ListNestedBlock{
				Description: "List of masking rules to apply to the branch.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"database_name": schema.StringAttribute{
							Description: "The name of the database containing the table to be masked.",
							Required:    true,
						},
						"schema_name": schema.StringAttribute{
							Description: "The name of the schema containing the table to be masked.",
							Required:    true,
						},
						"table_name": schema.StringAttribute{
							Description: "The name of the table containing the column to be masked.",
							Required:    true,
						},
						"column_name": schema.StringAttribute{
							Description: "The name of the column to be masked.",
							Required:    true,
						},
						"masking_function": schema.StringAttribute{
							Description: "The PostgreSQL Anonymizer masking function to apply (e.g., 'anon.random_string(10)', 'anon.fake_email()').",
							Optional:    true,
						},
						"masking_value": schema.StringAttribute{
							Description: "A literal value to set on the column when masking.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (r *maskingRulesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client
}

func (r *maskingRulesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data maskingRulesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var maskingRuleModels []maskingRuleModel
	resp.Diagnostics.Append(data.MaskingRules.ElementsAs(ctx, &maskingRuleModels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiMaskingRules := make([]neon.MaskingRule, len(maskingRuleModels))
	for i, rule := range maskingRuleModels {
		apiMaskingRules[i] = toAPIMaskingRule(rule)
	}

	result, err := r.client.UpdateMaskingRules(ctx, &neon.MaskingRulesUpdateRequest{
		MaskingRules: apiMaskingRules,
	}, neon.UpdateMaskingRulesParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create masking rules", err.Error())
		return
	}

	mapMaskingRulesToModel(ctx, result.MaskingRules, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *maskingRulesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data maskingRulesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetMaskingRules(ctx, neon.GetMaskingRulesParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		if neonerror.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read masking rules", err.Error())
		return
	}

	mapMaskingRulesToModel(ctx, result.MaskingRules, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *maskingRulesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan maskingRulesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var maskingRuleModels []maskingRuleModel
	resp.Diagnostics.Append(plan.MaskingRules.ElementsAs(ctx, &maskingRuleModels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiMaskingRules := make([]neon.MaskingRule, len(maskingRuleModels))
	for i, rule := range maskingRuleModels {
		apiMaskingRules[i] = toAPIMaskingRule(rule)
	}

	result, err := r.client.UpdateMaskingRules(ctx, &neon.MaskingRulesUpdateRequest{
		MaskingRules: apiMaskingRules,
	}, neon.UpdateMaskingRulesParams{
		ProjectID: plan.ProjectID.ValueString(),
		BranchID:  plan.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update masking rules", err.Error())
		return
	}

	mapMaskingRulesToModel(ctx, result.MaskingRules, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *maskingRulesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data maskingRulesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateMaskingRules(ctx, &neon.MaskingRulesUpdateRequest{
		MaskingRules: []neon.MaskingRule{},
	}, neon.UpdateMaskingRulesParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete masking rules", err.Error())
		return
	}
}

func (r *maskingRulesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: {project_id}/{branch_id}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("branch_id"), parts[1])...)
}

func toAPIMaskingRule(rule maskingRuleModel) neon.MaskingRule {
	apiRule := neon.MaskingRule{
		DatabaseName: rule.DatabaseName.ValueString(),
		SchemaName:   rule.SchemaName.ValueString(),
		TableName:    rule.TableName.ValueString(),
		ColumnName:   rule.ColumnName.ValueString(),
	}
	if !rule.MaskingFunction.IsNull() && !rule.MaskingFunction.IsUnknown() {
		apiRule.MaskingFunction = neon.NewOptString(rule.MaskingFunction.ValueString())
	}
	if !rule.MaskingValue.IsNull() && !rule.MaskingValue.IsUnknown() {
		apiRule.MaskingValue = neon.NewOptString(rule.MaskingValue.ValueString())
	}
	return apiRule
}

func fromAPIMaskingRule(rule neon.MaskingRule) maskingRuleModel {
	m := maskingRuleModel{
		DatabaseName: types.StringValue(rule.DatabaseName),
		SchemaName:   types.StringValue(rule.SchemaName),
		TableName:    types.StringValue(rule.TableName),
		ColumnName:   types.StringValue(rule.ColumnName),
	}
	if v, ok := rule.MaskingFunction.Get(); ok {
		m.MaskingFunction = types.StringValue(v)
	} else {
		m.MaskingFunction = types.StringNull()
	}
	if v, ok := rule.MaskingValue.Get(); ok {
		m.MaskingValue = types.StringValue(v)
	} else {
		m.MaskingValue = types.StringNull()
	}
	return m
}

func mapMaskingRulesToModel(ctx context.Context, apiRules []neon.MaskingRule, data *maskingRulesResourceModel, diagnostics *diag.Diagnostics) {
	maskingRuleValues := make([]maskingRuleModel, len(apiRules))
	for i, rule := range apiRules {
		maskingRuleValues[i] = fromAPIMaskingRule(rule)
	}

	maskingRulesList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: maskingRuleAttrTypes()}, maskingRuleValues)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return
	}
	data.MaskingRules = maskingRulesList
}
