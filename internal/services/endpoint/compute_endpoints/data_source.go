package compute_endpoints

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

type endpointsDataSource struct {
	client *neon.Client
}

type endpointsDataSourceModel struct {
	ProjectID types.String    `tfsdk:"project_id"`
	Endpoints []endpointModel `tfsdk:"endpoints"`
}

type endpointModel struct {
	ID                    types.String  `tfsdk:"id"`
	BranchID              types.String  `tfsdk:"branch_id"`
	Type                  types.String  `tfsdk:"type"`
	Name                  types.String  `tfsdk:"name"`
	Host                  types.String  `tfsdk:"host"`
	RegionID              types.String  `tfsdk:"region_id"`
	CurrentState          types.String  `tfsdk:"current_state"`
	AutoscalingLimitMinCu types.Float64 `tfsdk:"autoscaling_limit_min_cu"`
	AutoscalingLimitMaxCu types.Float64 `tfsdk:"autoscaling_limit_max_cu"`
	SuspendTimeoutSeconds types.Int64   `tfsdk:"suspend_timeout_seconds"`
	PoolerEnabled         types.Bool    `tfsdk:"pooler_enabled"`
	PoolerMode            types.String  `tfsdk:"pooler_mode"`
	Disabled              types.Bool    `tfsdk:"disabled"`
	PasswordlessAccess    types.Bool    `tfsdk:"passwordless_access"`
	ComputeProvisioner    types.String  `tfsdk:"compute_provisioner"`
	CreatedAt             types.String  `tfsdk:"created_at"`
	UpdatedAt             types.String  `tfsdk:"updated_at"`
}

func NewDataSource() datasource.DataSource {
	return &endpointsDataSource{}
}

func (d *endpointsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_endpoints"
}

func (d *endpointsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of endpoints for a Neon project.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The Neon project ID.",
				Required:    true,
			},
			"endpoints": schema.ListNestedAttribute{
				Description: "The list of endpoints.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The endpoint ID.",
							Computed:    true,
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
						"created_at": schema.StringAttribute{
							Description: "The creation timestamp, in RFC 3339 format.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "The last update timestamp, in RFC 3339 format.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *endpointsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *endpointsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data endpointsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.ListProjectEndpoints(ctx, neon.ListProjectEndpointsParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list endpoints", err.Error())
		return
	}

	data.Endpoints = make([]endpointModel, len(result.Endpoints))
	for i, ep := range result.Endpoints {
		m := endpointModel{
			ID:                    types.StringValue(ep.ID),
			BranchID:              types.StringValue(ep.BranchID),
			Type:                  types.StringValue(string(ep.Type)),
			Host:                  types.StringValue(ep.Host),
			RegionID:              types.StringValue(ep.RegionID),
			CurrentState:          types.StringValue(string(ep.CurrentState)),
			AutoscalingLimitMinCu: types.Float64Value(float64(ep.AutoscalingLimitMinCu)),
			AutoscalingLimitMaxCu: types.Float64Value(float64(ep.AutoscalingLimitMaxCu)),
			SuspendTimeoutSeconds: types.Int64Value(int64(ep.SuspendTimeoutSeconds)),
			PoolerEnabled:         types.BoolValue(ep.PoolerEnabled),
			PoolerMode:            types.StringValue(string(ep.PoolerMode)),
			Disabled:              types.BoolValue(ep.Disabled),
			PasswordlessAccess:    types.BoolValue(ep.PasswordlessAccess),
			ComputeProvisioner:    types.StringValue(string(ep.Provisioner)),
			CreatedAt:             types.StringValue(ep.CreatedAt.String()),
			UpdatedAt:             types.StringValue(ep.UpdatedAt.String()),
		}

		if v, ok := ep.Name.Get(); ok {
			m.Name = types.StringValue(v)
		} else {
			m.Name = types.StringNull()
		}

		data.Endpoints[i] = m
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
