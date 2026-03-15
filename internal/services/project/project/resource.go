package project

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonerror"
)

var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

// Attr type definitions for nested objects.
var preloadLibrariesAttrTypes = map[string]attr.Type{
	"use_defaults":      types.BoolType,
	"enabled_libraries": types.ListType{ElemType: types.StringType},
}

var quotaAttrTypes = map[string]attr.Type{
	"active_time_seconds":  types.Int64Type,
	"compute_time_seconds": types.Int64Type,
	"written_data_bytes":   types.Int64Type,
	"data_transfer_bytes":  types.Int64Type,
	"logical_size_bytes":   types.Int64Type,
}

var allowedIpsAttrTypes = map[string]attr.Type{
	"ips":                     types.ListType{ElemType: types.StringType},
	"protected_branches_only": types.BoolType,
}

var maintenanceWindowAttrTypes = map[string]attr.Type{
	"weekdays":   types.ListType{ElemType: types.Int64Type},
	"start_time": types.StringType,
	"end_time":   types.StringType,
}

var settingsAttrTypes = map[string]attr.Type{
	"quota":                      types.ObjectType{AttrTypes: quotaAttrTypes},
	"allowed_ips":                types.ObjectType{AttrTypes: allowedIpsAttrTypes},
	"enable_logical_replication": types.BoolType,
	"maintenance_window":         types.ObjectType{AttrTypes: maintenanceWindowAttrTypes},
	"block_public_connections":   types.BoolType,
	"block_vpc_connections":      types.BoolType,
	"audit_log_level":            types.StringType,
	"hipaa":                      types.BoolType,
	"preload_libraries":          types.ObjectType{AttrTypes: preloadLibrariesAttrTypes},
}

var defaultEndpointSettingsAttrTypes = map[string]attr.Type{
	"pg_settings":              types.MapType{ElemType: types.StringType},
	"pgbouncer_settings":       types.MapType{ElemType: types.StringType},
	"autoscaling_limit_min_cu": types.Float64Type,
	"autoscaling_limit_max_cu": types.Float64Type,
	"suspend_timeout_seconds":  types.Int64Type,
}

type projectResource struct {
	client *neon.Client
}

type projectResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	RegionID                types.String `tfsdk:"region_id"`
	PgVersion               types.Int32  `tfsdk:"pg_version"`
	HistoryRetentionSeconds types.Int32  `tfsdk:"history_retention_seconds"`
	StorePasswords          types.Bool   `tfsdk:"store_passwords"`
	OrgID                   types.String `tfsdk:"org_id"`
	Provisioner             types.String `tfsdk:"compute_provisioner"`
	DefaultEndpointSettings types.Object `tfsdk:"default_endpoint_settings"`
	Settings                types.Object `tfsdk:"settings"`
	CreatedAt               types.String `tfsdk:"created_at"`
	UpdatedAt               types.String `tfsdk:"updated_at"`
}

// Intermediate model structs for conversion.
type defaultEndpointSettingsModel struct {
	PgSettings            types.Map     `tfsdk:"pg_settings"`
	PgbouncerSettings     types.Map     `tfsdk:"pgbouncer_settings"`
	AutoscalingLimitMinCu types.Float64 `tfsdk:"autoscaling_limit_min_cu"`
	AutoscalingLimitMaxCu types.Float64 `tfsdk:"autoscaling_limit_max_cu"`
	SuspendTimeoutSeconds types.Int64   `tfsdk:"suspend_timeout_seconds"`
}

type projectSettingsModel struct {
	Quota                    types.Object `tfsdk:"quota"`
	AllowedIps               types.Object `tfsdk:"allowed_ips"`
	EnableLogicalReplication types.Bool   `tfsdk:"enable_logical_replication"`
	MaintenanceWindow        types.Object `tfsdk:"maintenance_window"`
	BlockPublicConnections   types.Bool   `tfsdk:"block_public_connections"`
	BlockVpcConnections      types.Bool   `tfsdk:"block_vpc_connections"`
	AuditLogLevel            types.String `tfsdk:"audit_log_level"`
	Hipaa                    types.Bool   `tfsdk:"hipaa"`
	PreloadLibraries         types.Object `tfsdk:"preload_libraries"`
}

type projectQuotaModel struct {
	ActiveTimeSeconds  types.Int64 `tfsdk:"active_time_seconds"`
	ComputeTimeSeconds types.Int64 `tfsdk:"compute_time_seconds"`
	WrittenDataBytes   types.Int64 `tfsdk:"written_data_bytes"`
	DataTransferBytes  types.Int64 `tfsdk:"data_transfer_bytes"`
	LogicalSizeBytes   types.Int64 `tfsdk:"logical_size_bytes"`
}

type allowedIpsModel struct {
	Ips                   types.List `tfsdk:"ips"`
	ProtectedBranchesOnly types.Bool `tfsdk:"protected_branches_only"`
}

type maintenanceWindowModel struct {
	Weekdays  types.List   `tfsdk:"weekdays"`
	StartTime types.String `tfsdk:"start_time"`
	EndTime   types.String `tfsdk:"end_time"`
}

type preloadLibrariesModel struct {
	UseDefaults      types.Bool `tfsdk:"use_defaults"`
	EnabledLibraries types.List `tfsdk:"enabled_libraries"`
}

func NewResource() resource.Resource {
	return &projectResource{}
}

func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Neon project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The project ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The project name.",
				Optional:    true,
				Computed:    true,
			},
			"region_id": schema.StringAttribute{
				Description: "The region identifier. Cannot be changed after creation.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pg_version": schema.Int32Attribute{
				Description: "The Postgres version.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.RequiresReplace(),
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"history_retention_seconds": schema.Int32Attribute{
				Description: "The number of seconds to retain the shared history for all branches.",
				Optional:    true,
				Computed:    true,
			},
			"store_passwords": schema.BoolAttribute{
				Description: "Whether passwords are stored for roles in the project.",
				Optional:    true,
				Computed:    true,
			},
			"org_id": schema.StringAttribute{
				Description: "The organization ID. If set, the project belongs to the specified organization. Cannot be changed after creation.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"compute_provisioner": schema.StringAttribute{
				Description: "The provisioner for the project. Cannot be changed after creation.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_endpoint_settings": schema.SingleNestedAttribute{
				Description: "Default endpoint settings for the project.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"pg_settings": schema.MapAttribute{
						Description: "A raw representation of Postgres settings.",
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
					},
					"pgbouncer_settings": schema.MapAttribute{
						Description: "A raw representation of PgBouncer settings.",
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
					},
					"autoscaling_limit_min_cu": schema.Float64Attribute{
						Description: "The minimum number of Compute Units.",
						Optional:    true,
						Computed:    true,
					},
					"autoscaling_limit_max_cu": schema.Float64Attribute{
						Description: "The maximum number of Compute Units.",
						Optional:    true,
						Computed:    true,
					},
					"suspend_timeout_seconds": schema.Int64Attribute{
						Description: "Duration of inactivity in seconds after which the compute endpoint is automatically suspended.",
						Optional:    true,
						Computed:    true,
					},
				},
			},
			"settings": schema.SingleNestedAttribute{
				Description: "Project settings.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"quota": schema.SingleNestedAttribute{
						Description: "Per-project consumption quota.",
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"active_time_seconds": schema.Int64Attribute{
								Description: "The total amount of wall-clock time allowed to be spent by the project's compute endpoints.",
								Optional:    true,
								Computed:    true,
							},
							"compute_time_seconds": schema.Int64Attribute{
								Description: "The total amount of CPU seconds allowed to be spent by the project's compute endpoints.",
								Optional:    true,
								Computed:    true,
							},
							"written_data_bytes": schema.Int64Attribute{
								Description: "Total amount of data written to all of a project's branches.",
								Optional:    true,
								Computed:    true,
							},
							"data_transfer_bytes": schema.Int64Attribute{
								Description: "Total amount of data transferred from all of a project's branches using the proxy.",
								Optional:    true,
								Computed:    true,
							},
							"logical_size_bytes": schema.Int64Attribute{
								Description: "Limit on the logical size of every project's branch.",
								Optional:    true,
								Computed:    true,
							},
						},
					},
					"allowed_ips": schema.SingleNestedAttribute{
						Description: "A list of IP addresses that are allowed to connect to the endpoint.",
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"ips": schema.ListAttribute{
								Description: "A list of allowed IP addresses.",
								Optional:    true,
								Computed:    true,
								ElementType: types.StringType,
							},
							"protected_branches_only": schema.BoolAttribute{
								Description: "If true, the list will be applied only to protected branches.",
								Optional:    true,
								Computed:    true,
							},
						},
					},
					"enable_logical_replication": schema.BoolAttribute{
						Description: "Sets wal_level=logical for all compute endpoints in this project.",
						Optional:    true,
						Computed:    true,
					},
					"maintenance_window": schema.SingleNestedAttribute{
						Description: "The maintenance window configuration.",
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"weekdays": schema.ListAttribute{
								Description: "A list of weekdays when the maintenance window is active (1=Monday, 7=Sunday).",
								Required:    true,
								ElementType: types.Int64Type,
							},
							"start_time": schema.StringAttribute{
								Description: "Start time of the maintenance window in HH:MM format (UTC).",
								Required:    true,
							},
							"end_time": schema.StringAttribute{
								Description: "End time of the maintenance window in HH:MM format (UTC).",
								Required:    true,
							},
						},
					},
					"block_public_connections": schema.BoolAttribute{
						Description: "When set, connections from the public internet are disallowed.",
						Optional:    true,
						Computed:    true,
					},
					"block_vpc_connections": schema.BoolAttribute{
						Description: "When set, connections using VPC endpoints are disallowed.",
						Optional:    true,
						Computed:    true,
					},
					"audit_log_level": schema.StringAttribute{
						Description: "The audit log level. One of: base, extended, full.",
						Optional:    true,
						Computed:    true,
					},
					"hipaa": schema.BoolAttribute{
						Description: "Whether HIPAA compliance is enabled for the project.",
						Optional:    true,
						Computed:    true,
					},
					"preload_libraries": schema.SingleNestedAttribute{
						Description: "Configuration for preloaded Postgres libraries.",
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"use_defaults": schema.BoolAttribute{
								Description: "Whether to use the default preloaded libraries.",
								Optional:    true,
								Computed:    true,
							},
							"enabled_libraries": schema.ListAttribute{
								Description: "A list of libraries to preload.",
								Optional:    true,
								Computed:    true,
								ElementType: types.StringType,
							},
						},
					},
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
			},
		},
	}
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &neon.ProjectCreateRequest{
		Project: neon.ProjectCreateRequestProject{},
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		apiReq.Project.Name = neon.NewOptString(data.Name.ValueString())
	}
	if !data.RegionID.IsNull() && !data.RegionID.IsUnknown() {
		apiReq.Project.RegionID = neon.NewOptString(data.RegionID.ValueString())
	}
	if !data.PgVersion.IsNull() && !data.PgVersion.IsUnknown() {
		apiReq.Project.PgVersion = neon.NewOptPgVersion(neon.PgVersion(data.PgVersion.ValueInt32()))
	}
	if !data.HistoryRetentionSeconds.IsNull() && !data.HistoryRetentionSeconds.IsUnknown() {
		apiReq.Project.HistoryRetentionSeconds = neon.NewOptInt32(data.HistoryRetentionSeconds.ValueInt32())
	}
	if !data.StorePasswords.IsNull() && !data.StorePasswords.IsUnknown() {
		apiReq.Project.StorePasswords = neon.NewOptBool(data.StorePasswords.ValueBool())
	}
	if !data.OrgID.IsNull() && !data.OrgID.IsUnknown() {
		apiReq.Project.OrgID = neon.NewOptString(data.OrgID.ValueString())
	}
	if !data.Provisioner.IsNull() && !data.Provisioner.IsUnknown() {
		apiReq.Project.Provisioner = neon.NewOptProvisioner(neon.Provisioner(data.Provisioner.ValueString()))
	}

	if !data.DefaultEndpointSettings.IsNull() && !data.DefaultEndpointSettings.IsUnknown() {
		des := buildDefaultEndpointSettingsRequest(ctx, data.DefaultEndpointSettings, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		apiReq.Project.DefaultEndpointSettings = neon.NewOptDefaultEndpointSettings(des)
	}

	if !data.Settings.IsNull() && !data.Settings.IsUnknown() {
		settings := buildProjectSettingsRequest(ctx, data.Settings, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		apiReq.Project.Settings = neon.NewOptProjectSettingsData(settings)
	}

	result, err := r.client.CreateProject(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create project", err.Error())
		return
	}

	mapProjectToModel(ctx, &result.Project, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetProject(ctx, neon.GetProjectParams{
		ProjectID: data.ID.ValueString(),
	})
	if err != nil {
		if neonerror.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read project", err.Error())
		return
	}

	mapProjectToModel(ctx, &result.Project, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &neon.ProjectUpdateRequest{
		Project: neon.ProjectUpdateRequestProject{},
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		apiReq.Project.Name = neon.NewOptString(data.Name.ValueString())
	}
	if !data.HistoryRetentionSeconds.IsNull() && !data.HistoryRetentionSeconds.IsUnknown() {
		apiReq.Project.HistoryRetentionSeconds = neon.NewOptInt32(data.HistoryRetentionSeconds.ValueInt32())
	}

	if !data.DefaultEndpointSettings.IsNull() && !data.DefaultEndpointSettings.IsUnknown() {
		des := buildDefaultEndpointSettingsRequest(ctx, data.DefaultEndpointSettings, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		apiReq.Project.DefaultEndpointSettings = neon.NewOptDefaultEndpointSettings(des)
	}

	if !data.Settings.IsNull() && !data.Settings.IsUnknown() {
		settings := buildProjectSettingsRequest(ctx, data.Settings, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		apiReq.Project.Settings = neon.NewOptProjectSettingsData(settings)
	}

	result, err := r.client.UpdateProject(ctx, apiReq, neon.UpdateProjectParams{
		ProjectID: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update project", err.Error())
		return
	}

	mapProjectToModel(ctx, &result.Project, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProject(ctx, neon.DeleteProjectParams{
		ProjectID: data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete project", err.Error())
		return
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildDefaultEndpointSettingsRequest(ctx context.Context, obj basetypes.ObjectValue, diags *diag.Diagnostics) neon.DefaultEndpointSettings {
	var m defaultEndpointSettingsModel
	diags.Append(obj.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return neon.DefaultEndpointSettings{}
	}

	des := neon.DefaultEndpointSettings{}
	if !m.AutoscalingLimitMinCu.IsNull() && !m.AutoscalingLimitMinCu.IsUnknown() {
		des.AutoscalingLimitMinCu = neon.NewOptComputeUnit(neon.ComputeUnit(m.AutoscalingLimitMinCu.ValueFloat64()))
	}
	if !m.AutoscalingLimitMaxCu.IsNull() && !m.AutoscalingLimitMaxCu.IsUnknown() {
		des.AutoscalingLimitMaxCu = neon.NewOptComputeUnit(neon.ComputeUnit(m.AutoscalingLimitMaxCu.ValueFloat64()))
	}
	if !m.SuspendTimeoutSeconds.IsNull() && !m.SuspendTimeoutSeconds.IsUnknown() {
		des.SuspendTimeoutSeconds = neon.NewOptSuspendTimeoutSeconds(neon.SuspendTimeoutSeconds(m.SuspendTimeoutSeconds.ValueInt64()))
	}
	if !m.PgSettings.IsNull() && !m.PgSettings.IsUnknown() {
		pgSettings := make(map[string]string)
		diags.Append(m.PgSettings.ElementsAs(ctx, &pgSettings, false)...)
		if !diags.HasError() {
			des.PgSettings = neon.NewOptPgSettingsData(neon.PgSettingsData(pgSettings))
		}
	}
	if !m.PgbouncerSettings.IsNull() && !m.PgbouncerSettings.IsUnknown() {
		pgbouncerSettings := make(map[string]string)
		diags.Append(m.PgbouncerSettings.ElementsAs(ctx, &pgbouncerSettings, false)...)
		if !diags.HasError() {
			des.PgbouncerSettings = neon.NewOptPgbouncerSettingsData(neon.PgbouncerSettingsData(pgbouncerSettings))
		}
	}
	return des
}

func buildProjectSettingsRequest(ctx context.Context, obj basetypes.ObjectValue, diags *diag.Diagnostics) neon.ProjectSettingsData {
	var m projectSettingsModel
	diags.Append(obj.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return neon.ProjectSettingsData{}
	}

	settings := neon.ProjectSettingsData{}

	if !m.EnableLogicalReplication.IsNull() && !m.EnableLogicalReplication.IsUnknown() {
		settings.EnableLogicalReplication = neon.NewOptBool(m.EnableLogicalReplication.ValueBool())
	}
	if !m.BlockPublicConnections.IsNull() && !m.BlockPublicConnections.IsUnknown() {
		settings.BlockPublicConnections = neon.NewOptBool(m.BlockPublicConnections.ValueBool())
	}
	if !m.BlockVpcConnections.IsNull() && !m.BlockVpcConnections.IsUnknown() {
		settings.BlockVpcConnections = neon.NewOptBool(m.BlockVpcConnections.ValueBool())
	}
	if !m.AuditLogLevel.IsNull() && !m.AuditLogLevel.IsUnknown() {
		settings.AuditLogLevel = neon.NewOptProjectAuditLogLevel(neon.ProjectAuditLogLevel(m.AuditLogLevel.ValueString()))
	}
	if !m.Hipaa.IsNull() && !m.Hipaa.IsUnknown() {
		settings.Hipaa = neon.NewOptBool(m.Hipaa.ValueBool())
	}

	if !m.Quota.IsNull() && !m.Quota.IsUnknown() {
		var qm projectQuotaModel
		diags.Append(m.Quota.As(ctx, &qm, basetypes.ObjectAsOptions{})...)
		quota := neon.ProjectQuota{}
		if !qm.ActiveTimeSeconds.IsNull() && !qm.ActiveTimeSeconds.IsUnknown() {
			quota.ActiveTimeSeconds = neon.NewOptInt64(qm.ActiveTimeSeconds.ValueInt64())
		}
		if !qm.ComputeTimeSeconds.IsNull() && !qm.ComputeTimeSeconds.IsUnknown() {
			quota.ComputeTimeSeconds = neon.NewOptInt64(qm.ComputeTimeSeconds.ValueInt64())
		}
		if !qm.WrittenDataBytes.IsNull() && !qm.WrittenDataBytes.IsUnknown() {
			quota.WrittenDataBytes = neon.NewOptInt64(qm.WrittenDataBytes.ValueInt64())
		}
		if !qm.DataTransferBytes.IsNull() && !qm.DataTransferBytes.IsUnknown() {
			quota.DataTransferBytes = neon.NewOptInt64(qm.DataTransferBytes.ValueInt64())
		}
		if !qm.LogicalSizeBytes.IsNull() && !qm.LogicalSizeBytes.IsUnknown() {
			quota.LogicalSizeBytes = neon.NewOptInt64(qm.LogicalSizeBytes.ValueInt64())
		}
		settings.Quota = neon.NewOptProjectQuota(quota)
	}

	if !m.AllowedIps.IsNull() && !m.AllowedIps.IsUnknown() {
		var aim allowedIpsModel
		diags.Append(m.AllowedIps.As(ctx, &aim, basetypes.ObjectAsOptions{})...)
		allowedIps := neon.AllowedIps{}
		if !aim.Ips.IsNull() && !aim.Ips.IsUnknown() {
			var ips []string
			diags.Append(aim.Ips.ElementsAs(ctx, &ips, false)...)
			allowedIps.Ips = ips
		}
		if !aim.ProtectedBranchesOnly.IsNull() && !aim.ProtectedBranchesOnly.IsUnknown() {
			allowedIps.ProtectedBranchesOnly = neon.NewOptBool(aim.ProtectedBranchesOnly.ValueBool())
		}
		settings.AllowedIps = neon.NewOptAllowedIps(allowedIps)
	}

	if !m.MaintenanceWindow.IsNull() && !m.MaintenanceWindow.IsUnknown() {
		var mwm maintenanceWindowModel
		diags.Append(m.MaintenanceWindow.As(ctx, &mwm, basetypes.ObjectAsOptions{})...)
		mw := neon.MaintenanceWindow{
			StartTime: mwm.StartTime.ValueString(),
			EndTime:   mwm.EndTime.ValueString(),
		}
		if !mwm.Weekdays.IsNull() && !mwm.Weekdays.IsUnknown() {
			var weekdays []int
			diags.Append(mwm.Weekdays.ElementsAs(ctx, &weekdays, false)...)
			mw.Weekdays = weekdays
		}
		settings.MaintenanceWindow = neon.NewOptMaintenanceWindow(mw)
	}

	if !m.PreloadLibraries.IsNull() && !m.PreloadLibraries.IsUnknown() {
		var plm preloadLibrariesModel
		diags.Append(m.PreloadLibraries.As(ctx, &plm, basetypes.ObjectAsOptions{})...)
		pl := neon.PreloadLibraries{}
		if !plm.UseDefaults.IsNull() && !plm.UseDefaults.IsUnknown() {
			pl.UseDefaults = neon.NewOptBool(plm.UseDefaults.ValueBool())
		}
		if !plm.EnabledLibraries.IsNull() && !plm.EnabledLibraries.IsUnknown() {
			var libs []string
			diags.Append(plm.EnabledLibraries.ElementsAs(ctx, &libs, false)...)
			pl.EnabledLibraries = libs
		}
		settings.PreloadLibraries = neon.NewOptPreloadLibraries(pl)
	}

	return settings
}

func mapProjectToModel(ctx context.Context, p *neon.Project, data *projectResourceModel, diags *diag.Diagnostics) {
	data.ID = types.StringValue(p.ID)
	data.Name = types.StringValue(p.Name)
	data.RegionID = types.StringValue(p.RegionID)
	data.PgVersion = types.Int32Value(int32(p.PgVersion))
	data.HistoryRetentionSeconds = types.Int32Value(p.HistoryRetentionSeconds)
	data.StorePasswords = types.BoolValue(p.StorePasswords)
	data.Provisioner = types.StringValue(string(p.Provisioner))
	data.CreatedAt = types.StringValue(p.CreatedAt.String())
	data.UpdatedAt = types.StringValue(p.UpdatedAt.String())

	if p.OrgID.IsSet() {
		data.OrgID = types.StringValue(p.OrgID.Value)
	} else {
		data.OrgID = types.StringNull()
	}

	// Default endpoint settings
	if p.DefaultEndpointSettings.IsSet() {
		des := p.DefaultEndpointSettings.Value
		m := defaultEndpointSettingsModel{
			AutoscalingLimitMinCu: types.Float64Null(),
			AutoscalingLimitMaxCu: types.Float64Null(),
			SuspendTimeoutSeconds: types.Int64Null(),
			PgSettings:            types.MapNull(types.StringType),
			PgbouncerSettings:     types.MapNull(types.StringType),
		}
		if des.AutoscalingLimitMinCu.IsSet() {
			m.AutoscalingLimitMinCu = types.Float64Value(float64(des.AutoscalingLimitMinCu.Value))
		}
		if des.AutoscalingLimitMaxCu.IsSet() {
			m.AutoscalingLimitMaxCu = types.Float64Value(float64(des.AutoscalingLimitMaxCu.Value))
		}
		if des.SuspendTimeoutSeconds.IsSet() {
			m.SuspendTimeoutSeconds = types.Int64Value(int64(des.SuspendTimeoutSeconds.Value))
		}
		if des.PgSettings.IsSet() {
			pgMap := make(map[string]attr.Value)
			for k, v := range des.PgSettings.Value {
				pgMap[k] = types.StringValue(v)
			}
			mapVal, d := types.MapValue(types.StringType, pgMap)
			diags.Append(d...)
			m.PgSettings = mapVal
		}
		if des.PgbouncerSettings.IsSet() {
			pgbMap := make(map[string]attr.Value)
			for k, v := range des.PgbouncerSettings.Value {
				pgbMap[k] = types.StringValue(v)
			}
			mapVal, d := types.MapValue(types.StringType, pgbMap)
			diags.Append(d...)
			m.PgbouncerSettings = mapVal
		}
		obj, d := types.ObjectValueFrom(ctx, defaultEndpointSettingsAttrTypes, m)
		diags.Append(d...)
		data.DefaultEndpointSettings = obj
	} else {
		data.DefaultEndpointSettings = types.ObjectNull(defaultEndpointSettingsAttrTypes)
	}

	// Settings
	if p.Settings.IsSet() {
		s := p.Settings.Value
		m := projectSettingsModel{
			EnableLogicalReplication: types.BoolNull(),
			BlockPublicConnections:   types.BoolNull(),
			BlockVpcConnections:      types.BoolNull(),
			AuditLogLevel:            types.StringNull(),
			Hipaa:                    types.BoolNull(),
			Quota:                    types.ObjectNull(quotaAttrTypes),
			AllowedIps:               types.ObjectNull(allowedIpsAttrTypes),
			MaintenanceWindow:        types.ObjectNull(maintenanceWindowAttrTypes),
			PreloadLibraries:         types.ObjectNull(preloadLibrariesAttrTypes),
		}

		if s.EnableLogicalReplication.IsSet() {
			m.EnableLogicalReplication = types.BoolValue(s.EnableLogicalReplication.Value)
		}
		if s.BlockPublicConnections.IsSet() {
			m.BlockPublicConnections = types.BoolValue(s.BlockPublicConnections.Value)
		}
		if s.BlockVpcConnections.IsSet() {
			m.BlockVpcConnections = types.BoolValue(s.BlockVpcConnections.Value)
		}
		if s.AuditLogLevel.IsSet() {
			m.AuditLogLevel = types.StringValue(string(s.AuditLogLevel.Value))
		}
		if s.Hipaa.IsSet() {
			m.Hipaa = types.BoolValue(s.Hipaa.Value)
		}

		if s.Quota.IsSet() {
			q := s.Quota.Value
			qm := projectQuotaModel{
				ActiveTimeSeconds:  types.Int64Null(),
				ComputeTimeSeconds: types.Int64Null(),
				WrittenDataBytes:   types.Int64Null(),
				DataTransferBytes:  types.Int64Null(),
				LogicalSizeBytes:   types.Int64Null(),
			}
			if q.ActiveTimeSeconds.IsSet() {
				qm.ActiveTimeSeconds = types.Int64Value(q.ActiveTimeSeconds.Value)
			}
			if q.ComputeTimeSeconds.IsSet() {
				qm.ComputeTimeSeconds = types.Int64Value(q.ComputeTimeSeconds.Value)
			}
			if q.WrittenDataBytes.IsSet() {
				qm.WrittenDataBytes = types.Int64Value(q.WrittenDataBytes.Value)
			}
			if q.DataTransferBytes.IsSet() {
				qm.DataTransferBytes = types.Int64Value(q.DataTransferBytes.Value)
			}
			if q.LogicalSizeBytes.IsSet() {
				qm.LogicalSizeBytes = types.Int64Value(q.LogicalSizeBytes.Value)
			}
			obj, d := types.ObjectValueFrom(ctx, quotaAttrTypes, qm)
			diags.Append(d...)
			m.Quota = obj
		}

		if s.AllowedIps.IsSet() {
			ai := s.AllowedIps.Value
			aim := allowedIpsModel{
				Ips:                   types.ListNull(types.StringType),
				ProtectedBranchesOnly: types.BoolNull(),
			}
			if ai.Ips != nil {
				ipValues := make([]attr.Value, len(ai.Ips))
				for i, ip := range ai.Ips {
					ipValues[i] = types.StringValue(ip)
				}
				listVal, d := types.ListValue(types.StringType, ipValues)
				diags.Append(d...)
				aim.Ips = listVal
			}
			if ai.ProtectedBranchesOnly.IsSet() {
				aim.ProtectedBranchesOnly = types.BoolValue(ai.ProtectedBranchesOnly.Value)
			}
			obj, d := types.ObjectValueFrom(ctx, allowedIpsAttrTypes, aim)
			diags.Append(d...)
			m.AllowedIps = obj
		}

		if s.MaintenanceWindow.IsSet() {
			mw := s.MaintenanceWindow.Value
			mwm := maintenanceWindowModel{
				StartTime: types.StringValue(mw.StartTime),
				EndTime:   types.StringValue(mw.EndTime),
				Weekdays:  types.ListNull(types.Int64Type),
			}
			if mw.Weekdays != nil {
				wdValues := make([]attr.Value, len(mw.Weekdays))
				for i, wd := range mw.Weekdays {
					wdValues[i] = types.Int64Value(int64(wd))
				}
				listVal, d := types.ListValue(types.Int64Type, wdValues)
				diags.Append(d...)
				mwm.Weekdays = listVal
			}
			obj, d := types.ObjectValueFrom(ctx, maintenanceWindowAttrTypes, mwm)
			diags.Append(d...)
			m.MaintenanceWindow = obj
		}

		if s.PreloadLibraries.IsSet() {
			pl := s.PreloadLibraries.Value
			plm := preloadLibrariesModel{
				UseDefaults:      types.BoolNull(),
				EnabledLibraries: types.ListNull(types.StringType),
			}
			if pl.UseDefaults.IsSet() {
				plm.UseDefaults = types.BoolValue(pl.UseDefaults.Value)
			}
			if pl.EnabledLibraries != nil {
				libValues := make([]attr.Value, len(pl.EnabledLibraries))
				for i, lib := range pl.EnabledLibraries {
					libValues[i] = types.StringValue(lib)
				}
				listVal, d := types.ListValue(types.StringType, libValues)
				diags.Append(d...)
				plm.EnabledLibraries = listVal
			}
			obj, d := types.ObjectValueFrom(ctx, preloadLibrariesAttrTypes, plm)
			diags.Append(d...)
			m.PreloadLibraries = obj
		}

		obj, d := types.ObjectValueFrom(ctx, settingsAttrTypes, m)
		diags.Append(d...)
		data.Settings = obj
	} else {
		data.Settings = types.ObjectNull(settingsAttrTypes)
	}
}
