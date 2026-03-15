package project

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type projectDataSource struct {
	client *neon.Client
}

func NewDataSource() datasource.DataSource {
	return &projectDataSource{}
}

func (d *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a Neon project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The project ID.",
				Required:    true,
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
			"history_retention_seconds": schema.Int32Attribute{
				Description: "The number of seconds to retain the shared history for all branches.",
				Computed:    true,
			},
			"store_passwords": schema.BoolAttribute{
				Description: "Whether passwords are stored for roles in the project.",
				Computed:    true,
			},
			"org_id": schema.StringAttribute{
				Description: "The organization ID.",
				Computed:    true,
			},
			"compute_provisioner": schema.StringAttribute{
				Description: "The provisioner for the project.",
				Computed:    true,
			},
			"default_endpoint_settings": schema.SingleNestedAttribute{
				Description: "Default endpoint settings for the project.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"pg_settings": schema.MapAttribute{
						Description: "A raw representation of Postgres settings.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"pgbouncer_settings": schema.MapAttribute{
						Description: "A raw representation of PgBouncer settings.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"autoscaling_limit_min_cu": schema.Float64Attribute{
						Description: "The minimum number of Compute Units.",
						Computed:    true,
					},
					"autoscaling_limit_max_cu": schema.Float64Attribute{
						Description: "The maximum number of Compute Units.",
						Computed:    true,
					},
					"suspend_timeout_seconds": schema.Int64Attribute{
						Description: "Duration of inactivity in seconds after which the compute endpoint is automatically suspended.",
						Computed:    true,
					},
				},
			},
			"settings": schema.SingleNestedAttribute{
				Description: "Project settings.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"quota": schema.SingleNestedAttribute{
						Description: "Per-project consumption quota.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"active_time_seconds":  schema.Int64Attribute{Computed: true},
							"compute_time_seconds": schema.Int64Attribute{Computed: true},
							"written_data_bytes":   schema.Int64Attribute{Computed: true},
							"data_transfer_bytes":  schema.Int64Attribute{Computed: true},
							"logical_size_bytes":   schema.Int64Attribute{Computed: true},
						},
					},
					"allowed_ips": schema.SingleNestedAttribute{
						Description: "Allowed IP addresses configuration.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"ips":                     schema.ListAttribute{Computed: true, ElementType: types.StringType},
							"protected_branches_only": schema.BoolAttribute{Computed: true},
						},
					},
					"enable_logical_replication": schema.BoolAttribute{Computed: true},
					"maintenance_window": schema.SingleNestedAttribute{
						Computed: true,
						Attributes: map[string]schema.Attribute{
							"weekdays":   schema.ListAttribute{Computed: true, ElementType: types.Int64Type},
							"start_time": schema.StringAttribute{Computed: true},
							"end_time":   schema.StringAttribute{Computed: true},
						},
					},
					"block_public_connections": schema.BoolAttribute{Computed: true},
					"block_vpc_connections":    schema.BoolAttribute{Computed: true},
					"audit_log_level":          schema.StringAttribute{Computed: true},
					"hipaa":                    schema.BoolAttribute{Computed: true},
					"preload_libraries": schema.SingleNestedAttribute{
						Computed: true,
						Attributes: map[string]schema.Attribute{
							"use_defaults":      schema.BoolAttribute{Computed: true},
							"enabled_libraries": schema.ListAttribute{Computed: true, ElementType: types.StringType},
						},
					},
				},
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
	}
}

func (d *projectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data projectResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetProject(ctx, neon.GetProjectParams{
		ProjectID: data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read project", err.Error())
		return
	}

	mapProjectToModel(ctx, &result.Project, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
