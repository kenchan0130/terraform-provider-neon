package compute_endpoint

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type endpointDataSource struct {
	client *neon.Client
}

func NewDataSource() datasource.DataSource {
	return &endpointDataSource{}
}

func (d *endpointDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_endpoint"
}

func (d *endpointDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a Neon endpoint.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The endpoint ID.",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The project ID.",
				Required:    true,
			},
			"branch_id": schema.StringAttribute{
				Description: "The branch ID.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The endpoint type.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Optional name of the compute endpoint.",
				Computed:    true,
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
				Description: "The duration of inactivity in seconds after which the compute is suspended.",
				Computed:    true,
			},
			"pooler_enabled": schema.BoolAttribute{
				Description: "Whether connection pooling is enabled.",
				Computed:    true,
			},
			"pooler_mode": schema.StringAttribute{
				Description: "The connection pooler mode.",
				Computed:    true,
			},
			"disabled": schema.BoolAttribute{
				Description: "Whether the endpoint is disabled.",
				Computed:    true,
			},
			"passwordless_access": schema.BoolAttribute{
				Description: "Whether to permit passwordless access to the compute endpoint.",
				Computed:    true,
			},
			"compute_provisioner": schema.StringAttribute{
				Description: "The provisioner for the compute endpoint.",
				Computed:    true,
			},
			"settings": schema.SingleNestedAttribute{
				Description: "Endpoint settings.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"pg_settings": schema.MapAttribute{
						Description: "A raw representation of Postgres settings.",
						ElementType: types.StringType,
						Computed:    true,
					},
					"pgbouncer_settings": schema.MapAttribute{
						Description: "A raw representation of PgBouncer settings.",
						ElementType: types.StringType,
						Computed:    true,
					},
					"preload_libraries": schema.SingleNestedAttribute{
						Description: "Preload libraries configuration.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"use_defaults": schema.BoolAttribute{
								Description: "Whether to use default preload libraries.",
								Computed:    true,
							},
							"enabled_libraries": schema.ListAttribute{
								Description: "List of enabled preload libraries.",
								ElementType: types.StringType,
								Computed:    true,
							},
						},
					},
				},
			},
			"host": schema.StringAttribute{
				Description: "The hostname for connecting to the endpoint.",
				Computed:    true,
			},
			"region_id": schema.StringAttribute{
				Description: "The region identifier.",
				Computed:    true,
			},
			"current_state": schema.StringAttribute{
				Description: "The current state of the compute endpoint.",
				Computed:    true,
			},
			"last_active": schema.StringAttribute{
				Description: "A timestamp indicating when the compute endpoint was last active.",
				Computed:    true,
			},
			"creation_source": schema.StringAttribute{
				Description: "The compute endpoint creation source.",
				Computed:    true,
			},
			"compute_release_version": schema.StringAttribute{
				Description: "Attached compute's release version number.",
				Computed:    true,
			},
			"pending_state": schema.StringAttribute{
				Description: "The pending state of the compute endpoint.",
				Computed:    true,
			},
			"started_at": schema.StringAttribute{
				Description: "A timestamp indicating when the compute endpoint was last started.",
				Computed:    true,
			},
			"suspended_at": schema.StringAttribute{
				Description: "A timestamp indicating when the compute endpoint was last suspended.",
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
	}
}

func (d *endpointDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *endpointDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data endpointResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetProjectEndpoint(ctx, neon.GetProjectEndpointParams{
		ProjectID:  data.ProjectID.ValueString(),
		EndpointID: data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read endpoint", err.Error())
		return
	}

	mapEndpointToModel(ctx, &result.Endpoint, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
