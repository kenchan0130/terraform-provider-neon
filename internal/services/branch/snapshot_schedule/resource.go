package snapshot_schedule

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonerror"
)

var (
	_ resource.Resource                = &snapshotScheduleResource{}
	_ resource.ResourceWithConfigure   = &snapshotScheduleResource{}
	_ resource.ResourceWithImportState = &snapshotScheduleResource{}
)

type snapshotScheduleResource struct {
	client *neon.Client
}

type snapshotScheduleResourceModel struct {
	ProjectID types.String `tfsdk:"project_id"`
	BranchID  types.String `tfsdk:"branch_id"`
	Schedule  types.List   `tfsdk:"schedule"`
}

type scheduleItemModel struct {
	Frequency        types.String `tfsdk:"frequency"`
	Hour             types.Int64  `tfsdk:"hour"`
	Day              types.Int64  `tfsdk:"day"`
	Month            types.Int64  `tfsdk:"month"`
	RetentionSeconds types.Int64  `tfsdk:"retention_seconds"`
}

func scheduleItemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"frequency":         types.StringType,
		"hour":              types.Int64Type,
		"day":               types.Int64Type,
		"month":             types.Int64Type,
		"retention_seconds": types.Int64Type,
	}
}

func NewResource() resource.Resource {
	return &snapshotScheduleResource{}
}

func (r *snapshotScheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot_schedule"
}

func (r *snapshotScheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Neon snapshot schedule.",
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
			"schedule": schema.ListNestedBlock{
				Description: "List of snapshot schedule entries.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"frequency": schema.StringAttribute{
							Description: "How often to take snapshots. Must be one of: hourly, daily, weekly, monthly, yearly.",
							Required:    true,
						},
						"hour": schema.Int64Attribute{
							Description: "The hour of the day to take the snapshot (if applicable).",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"day": schema.Int64Attribute{
							Description: "The day of the week or month to take the snapshot (if applicable).",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"month": schema.Int64Attribute{
							Description: "The month of the year to take the snapshot (if applicable).",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"retention_seconds": schema.Int64Attribute{
							Description: "How long to keep a snapshot (in seconds) before it's automatically deleted.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
		},
	}
}

func (r *snapshotScheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *snapshotScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data snapshotScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var scheduleModels []scheduleItemModel
	resp.Diagnostics.Append(data.Schedule.ElementsAs(ctx, &scheduleModels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiSchedule := toAPISchedule(scheduleModels)

	err := r.client.SetSnapshotSchedule(ctx, &neon.BackupSchedule{
		Schedule: apiSchedule,
	}, neon.SetSnapshotScheduleParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create snapshot schedule", err.Error())
		return
	}

	// Read back to get full state.
	result, err := r.client.GetSnapshotSchedule(ctx, neon.GetSnapshotScheduleParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read snapshot schedule after create", err.Error())
		return
	}

	mapScheduleToModel(ctx, result.Schedule, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *snapshotScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data snapshotScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetSnapshotSchedule(ctx, neon.GetSnapshotScheduleParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		if neonerror.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read snapshot schedule", err.Error())
		return
	}

	mapScheduleToModel(ctx, result.Schedule, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *snapshotScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan snapshotScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var scheduleModels []scheduleItemModel
	resp.Diagnostics.Append(plan.Schedule.ElementsAs(ctx, &scheduleModels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiSchedule := toAPISchedule(scheduleModels)

	err := r.client.SetSnapshotSchedule(ctx, &neon.BackupSchedule{
		Schedule: apiSchedule,
	}, neon.SetSnapshotScheduleParams{
		ProjectID: plan.ProjectID.ValueString(),
		BranchID:  plan.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update snapshot schedule", err.Error())
		return
	}

	// Read back to get full state.
	result, err := r.client.GetSnapshotSchedule(ctx, neon.GetSnapshotScheduleParams{
		ProjectID: plan.ProjectID.ValueString(),
		BranchID:  plan.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read snapshot schedule after update", err.Error())
		return
	}

	mapScheduleToModel(ctx, result.Schedule, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *snapshotScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data snapshotScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.SetSnapshotSchedule(ctx, &neon.BackupSchedule{
		Schedule: []neon.BackupScheduleItem{},
	}, neon.SetSnapshotScheduleParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete snapshot schedule", err.Error())
		return
	}
}

func (r *snapshotScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func toAPISchedule(models []scheduleItemModel) []neon.BackupScheduleItem {
	items := make([]neon.BackupScheduleItem, len(models))
	for i, m := range models {
		items[i] = neon.BackupScheduleItem{
			Frequency: m.Frequency.ValueString(),
		}
		if !m.Hour.IsNull() && !m.Hour.IsUnknown() {
			items[i].Hour = neon.NewOptInt(int(m.Hour.ValueInt64()))
		}
		if !m.Day.IsNull() && !m.Day.IsUnknown() {
			items[i].Day = neon.NewOptInt(int(m.Day.ValueInt64()))
		}
		if !m.Month.IsNull() && !m.Month.IsUnknown() {
			items[i].Month = neon.NewOptInt(int(m.Month.ValueInt64()))
		}
		if !m.RetentionSeconds.IsNull() && !m.RetentionSeconds.IsUnknown() {
			items[i].RetentionSeconds = neon.NewOptInt(int(m.RetentionSeconds.ValueInt64()))
		}
	}
	return items
}

func fromAPIScheduleItem(item neon.BackupScheduleItem) scheduleItemModel {
	m := scheduleItemModel{
		Frequency: types.StringValue(item.Frequency),
	}
	if v, ok := item.Hour.Get(); ok {
		m.Hour = types.Int64Value(int64(v))
	} else {
		m.Hour = types.Int64Null()
	}
	if v, ok := item.Day.Get(); ok {
		m.Day = types.Int64Value(int64(v))
	} else {
		m.Day = types.Int64Null()
	}
	if v, ok := item.Month.Get(); ok {
		m.Month = types.Int64Value(int64(v))
	} else {
		m.Month = types.Int64Null()
	}
	if v, ok := item.RetentionSeconds.Get(); ok {
		m.RetentionSeconds = types.Int64Value(int64(v))
	} else {
		m.RetentionSeconds = types.Int64Null()
	}
	return m
}

func mapScheduleToModel(ctx context.Context, apiItems []neon.BackupScheduleItem, data *snapshotScheduleResourceModel, diagnostics *diag.Diagnostics) {
	scheduleValues := make([]scheduleItemModel, len(apiItems))
	for i, item := range apiItems {
		scheduleValues[i] = fromAPIScheduleItem(item)
	}

	scheduleList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: scheduleItemAttrTypes()}, scheduleValues)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return
	}
	data.Schedule = scheduleList
}
