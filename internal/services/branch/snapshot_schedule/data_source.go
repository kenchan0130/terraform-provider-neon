package snapshot_schedule

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type snapshotScheduleDataSource struct {
	client *neon.Client
}

type snapshotScheduleDataSourceModel struct {
	ProjectID types.String `tfsdk:"project_id"`
	BranchID  types.String `tfsdk:"branch_id"`
	Schedule  types.List   `tfsdk:"schedule"`
}

func NewDataSource() datasource.DataSource {
	return &snapshotScheduleDataSource{}
}

func (d *snapshotScheduleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot_schedule"
}

func (d *snapshotScheduleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a Neon snapshot schedule.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID.",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"schedule": schema.ListNestedBlock{
				Description: "List of snapshot schedule entries.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"frequency": schema.StringAttribute{
							Description: "How often to take snapshots.",
							Computed:    true,
						},
						"hour": schema.Int64Attribute{
							Description: "The hour of the day to take the snapshot.",
							Computed:    true,
						},
						"day": schema.Int64Attribute{
							Description: "The day of the week or month to take the snapshot.",
							Computed:    true,
						},
						"month": schema.Int64Attribute{
							Description: "The month of the year to take the snapshot.",
							Computed:    true,
						},
						"retention_seconds": schema.Int64Attribute{
							Description: "How long to keep a snapshot (in seconds) before it's automatically deleted.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *snapshotScheduleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *snapshotScheduleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data snapshotScheduleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetSnapshotSchedule(ctx, neon.GetSnapshotScheduleParams{
		ProjectID: data.ProjectID.ValueString(),
		BranchID:  data.BranchID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read snapshot schedule", err.Error())
		return
	}

	scheduleValues := make([]scheduleItemModel, len(result.Schedule))
	for i, item := range result.Schedule {
		scheduleValues[i] = fromAPIScheduleItem(item)
	}

	scheduleList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: scheduleItemAttrTypes()}, scheduleValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Schedule = scheduleList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
