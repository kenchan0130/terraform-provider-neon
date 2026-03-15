package snapshot

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type snapshotDataSource struct {
	client *neon.Client
}

type snapshotDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectID      types.String `tfsdk:"project_id"`
	Name           types.String `tfsdk:"name"`
	Lsn            types.String `tfsdk:"lsn"`
	Timestamp      types.String `tfsdk:"timestamp"`
	SourceBranchID types.String `tfsdk:"source_branch_id"`
	ExpiresAt      types.String `tfsdk:"expires_at"`
	Manual         types.Bool   `tfsdk:"manual"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

func NewDataSource() datasource.DataSource {
	return &snapshotDataSource{}
}

func (d *snapshotDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot"
}

func (d *snapshotDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a Neon snapshot.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The snapshot ID.",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The snapshot name.",
				Computed:    true,
			},
			"lsn": schema.StringAttribute{
				Description: "The Log Sequence Number (LSN) of the snapshot.",
				Computed:    true,
			},
			"timestamp": schema.StringAttribute{
				Description: "The snapshot timestamp.",
				Computed:    true,
			},
			"source_branch_id": schema.StringAttribute{
				Description: "The ID of the branch the snapshot was created from.",
				Computed:    true,
			},
			"expires_at": schema.StringAttribute{
				Description: "The time at which the snapshot will be automatically deleted.",
				Computed:    true,
			},
			"manual": schema.BoolAttribute{
				Description: "Whether the snapshot was manually created.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
			},
		},
	}
}

func (d *snapshotDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *snapshotDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data snapshotDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.ListSnapshots(ctx, neon.ListSnapshotsParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list snapshots", err.Error())
		return
	}

	for i := range result.Snapshots {
		if result.Snapshots[i].ID == data.ID.ValueString() {
			s := &result.Snapshots[i]
			data.Name = types.StringValue(s.Name)
			data.CreatedAt = types.StringValue(s.CreatedAt)

			if v, ok := s.Lsn.Get(); ok {
				data.Lsn = types.StringValue(v)
			} else {
				data.Lsn = types.StringNull()
			}

			if v, ok := s.Timestamp.Get(); ok {
				data.Timestamp = types.StringValue(v)
			} else {
				data.Timestamp = types.StringNull()
			}

			if v, ok := s.SourceBranchID.Get(); ok {
				data.SourceBranchID = types.StringValue(v)
			} else {
				data.SourceBranchID = types.StringNull()
			}

			if v, ok := s.ExpiresAt.Get(); ok {
				data.ExpiresAt = types.StringValue(v)
			} else {
				data.ExpiresAt = types.StringNull()
			}

			if v, ok := s.Manual.Get(); ok {
				data.Manual = types.BoolValue(v)
			} else {
				data.Manual = types.BoolNull()
			}

			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"Snapshot Not Found",
		fmt.Sprintf("Snapshot %q not found in project %q.", data.ID.ValueString(), data.ProjectID.ValueString()),
	)
}
