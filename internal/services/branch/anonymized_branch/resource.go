package anonymized_branch

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
	_ resource.Resource                = &anonymizedBranchResource{}
	_ resource.ResourceWithConfigure   = &anonymizedBranchResource{}
	_ resource.ResourceWithImportState = &anonymizedBranchResource{}
)

type anonymizedBranchResource struct {
	client *neon.Client
}

type anonymizedBranchResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectID          types.String `tfsdk:"project_id"`
	ParentID           types.String `tfsdk:"parent_id"`
	Name               types.String `tfsdk:"name"`
	StartAnonymization types.Bool   `tfsdk:"start_anonymization"`
	MaskingRules       types.List   `tfsdk:"masking_rules"`
	State              types.String `tfsdk:"state"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
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
	return &anonymizedBranchResource{}
}

func (r *anonymizedBranchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_anonymized_branch"
}

func (r *anonymizedBranchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Neon anonymized branch with masking rules.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The branch ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "The project ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent_id": schema.StringAttribute{
				Description: "The parent branch ID. If omitted, the branch will be created from the project's default branch.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The branch name.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"start_anonymization": schema.BoolAttribute{
				Description: "If true, automatically start anonymization. Changing from true to false requires resource replacement.",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					startAnonymizationPlanModifier{},
				},
			},
			"state": schema.StringAttribute{
				Description: "The current state of the anonymized branch (e.g., created, initialized, anonymizing, anonymized, error).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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

func (r *anonymizedBranchResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *anonymizedBranchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data anonymizedBranchResourceModel
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

	branchReq := neon.BranchCreateRequestBranch{}
	if !data.ParentID.IsNull() && !data.ParentID.IsUnknown() {
		branchReq.ParentID = neon.NewOptString(data.ParentID.ValueString())
	}
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		branchReq.Name = neon.NewOptString(data.Name.ValueString())
	}

	createReq := &neon.BranchAnonymizedCreateRequest{
		BranchCreate: neon.NewOptBranchCreateRequest(neon.BranchCreateRequest{
			Branch: neon.NewOptBranchCreateRequestBranch(branchReq),
		}),
		MaskingRules: apiMaskingRules,
	}

	if !data.StartAnonymization.IsNull() && !data.StartAnonymization.IsUnknown() {
		createReq.StartAnonymization = neon.NewOptBool(data.StartAnonymization.ValueBool())
	}

	result, err := r.client.CreateProjectBranchAnonymized(ctx, createReq, neon.CreateProjectBranchAnonymizedParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create anonymized branch", err.Error())
		return
	}

	data.ID = types.StringValue(result.Branch.ID)
	data.ProjectID = types.StringValue(result.Branch.ProjectID)
	data.Name = types.StringValue(result.Branch.Name)

	if v, ok := result.Branch.ParentID.Get(); ok {
		data.ParentID = types.StringValue(v)
	} else {
		data.ParentID = types.StringNull()
	}

	r.readState(ctx, &data, resp)
}

func (r *anonymizedBranchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data anonymizedBranchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if the branch still exists before reading full state.
	_, err := r.client.GetProjectBranch(ctx, neon.GetProjectBranchParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.ID.ValueString(),
	})
	if err != nil {
		if neonerror.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
	}

	r.readState(ctx, &data, resp)
}

func (r *anonymizedBranchResource) readState(ctx context.Context, data *anonymizedBranchResourceModel, resp any) {
	diagsPtr := getDiagnostics(resp)
	projectID := data.ProjectID.ValueString()
	branchID := data.ID.ValueString()

	branchResult, err := r.client.GetProjectBranch(ctx, neon.GetProjectBranchParams{
		ProjectID: projectID,
		BranchID:  branchID,
	})
	if err != nil {
		diagsPtr.AddError("Failed to read branch", err.Error())
		return
	}

	data.Name = types.StringValue(branchResult.Branch.Name)
	if v, ok := branchResult.Branch.ParentID.Get(); ok {
		data.ParentID = types.StringValue(v)
	} else {
		data.ParentID = types.StringNull()
	}

	maskingResult, err := r.client.GetMaskingRules(ctx, neon.GetMaskingRulesParams{
		ProjectID: projectID,
		BranchID:  branchID,
	})
	if err != nil {
		diagsPtr.AddError("Failed to read masking rules", err.Error())
		return
	}

	maskingRuleValues := make([]maskingRuleModel, len(maskingResult.MaskingRules))
	for i, rule := range maskingResult.MaskingRules {
		maskingRuleValues[i] = fromAPIMaskingRule(rule)
	}

	maskingRulesList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: maskingRuleAttrTypes()}, maskingRuleValues)
	diagsPtr.Append(diags...)
	if diags.HasError() {
		return
	}
	data.MaskingRules = maskingRulesList

	statusResult, err := r.client.GetAnonymizedBranchStatus(ctx, neon.GetAnonymizedBranchStatusParams{
		ProjectID: projectID,
		BranchID:  branchID,
	})
	if err != nil {
		diagsPtr.AddError("Failed to read anonymized branch status", err.Error())
		return
	}

	data.State = types.StringValue(statusResult.State)
	data.CreatedAt = types.StringValue(statusResult.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	data.UpdatedAt = types.StringValue(statusResult.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	setStateFromResp(ctx, resp, data)
}

func getDiagnostics(resp any) *diag.Diagnostics {
	switch v := resp.(type) {
	case *resource.CreateResponse:
		return &v.Diagnostics
	case *resource.ReadResponse:
		return &v.Diagnostics
	case *resource.UpdateResponse:
		return &v.Diagnostics
	default:
		return nil
	}
}

func setStateFromResp(ctx context.Context, resp any, data *anonymizedBranchResourceModel) {
	switch v := resp.(type) {
	case *resource.CreateResponse:
		v.Diagnostics.Append(v.State.Set(ctx, data)...)
	case *resource.ReadResponse:
		v.Diagnostics.Append(v.State.Set(ctx, data)...)
	case *resource.UpdateResponse:
		v.Diagnostics.Append(v.State.Set(ctx, data)...)
	}
}

func (r *anonymizedBranchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan anonymizedBranchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state anonymizedBranchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	branchID := state.ID.ValueString()

	if !plan.Name.Equal(state.Name) {
		updateReq := &neon.BranchUpdateRequest{
			Branch: neon.BranchUpdateRequestBranch{},
		}
		if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
			updateReq.Branch.Name = neon.NewOptString(plan.Name.ValueString())
		}

		_, err := r.client.UpdateProjectBranch(ctx, updateReq, neon.UpdateProjectBranchParams{
			ProjectID: projectID,
			BranchID:  branchID,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update branch name", err.Error())
			return
		}
	}

	if !plan.MaskingRules.Equal(state.MaskingRules) {
		var maskingRuleModels []maskingRuleModel
		resp.Diagnostics.Append(plan.MaskingRules.ElementsAs(ctx, &maskingRuleModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		apiMaskingRules := make([]neon.MaskingRule, len(maskingRuleModels))
		for i, rule := range maskingRuleModels {
			apiMaskingRules[i] = toAPIMaskingRule(rule)
		}

		_, err := r.client.UpdateMaskingRules(ctx, &neon.MaskingRulesUpdateRequest{
			MaskingRules: apiMaskingRules,
		}, neon.UpdateMaskingRulesParams{
			ProjectID: projectID,
			BranchID:  branchID,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update masking rules", err.Error())
			return
		}
	}

	if !plan.StartAnonymization.IsNull() && plan.StartAnonymization.ValueBool() &&
		(state.StartAnonymization.IsNull() || !state.StartAnonymization.ValueBool()) {
		_, err := r.client.StartAnonymization(ctx, neon.StartAnonymizationParams{
			ProjectID: projectID,
			BranchID:  branchID,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to start anonymization", err.Error())
			return
		}
	}

	plan.ID = state.ID
	plan.ProjectID = state.ProjectID
	r.readState(ctx, &plan, resp)
}

func (r *anonymizedBranchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data anonymizedBranchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectBranch(ctx, neon.DeleteProjectBranchParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete anonymized branch", err.Error())
		return
	}
}

func (r *anonymizedBranchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: {project_id}/{branch_id}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
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

// startAnonymizationPlanModifier implements a custom plan modifier for the start_anonymization attribute.
// false → true: in-place update (StartAnonymization API call)
// true → false: RequiresReplace (resource must be recreated).
type startAnonymizationPlanModifier struct{}

func (m startAnonymizationPlanModifier) Description(_ context.Context) string {
	return "Changing start_anonymization from true to false requires resource replacement."
}

func (m startAnonymizationPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m startAnonymizationPlanModifier) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() {
		return
	}

	if req.StateValue.ValueBool() && !req.PlanValue.ValueBool() {
		resp.RequiresReplace = true
	}
}
