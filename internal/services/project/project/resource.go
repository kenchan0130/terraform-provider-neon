package project

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonerror"
	"github.com/kenchan0130/terraform-provider-neon/internal/planmodifiers"
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
		Attributes:  projectSchemaAttributes(),
	}
}

func projectSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
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
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
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
			PlanModifiers: []planmodifier.Int32{
				int32planmodifier.UseStateForUnknown(),
			},
		},
		"store_passwords": schema.BoolAttribute{
			Description: "Whether passwords are stored for roles in the project. Cannot be changed after creation.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.RequiresReplace(),
				boolplanmodifier.UseStateForUnknown(),
			},
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
		"default_endpoint_settings": defaultEndpointSettingsSchema(),
		"settings":                  projectSettingsSchema(),
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
				planmodifiers.UnknownOnResourceChange(),
			},
		},
	}
}

func defaultEndpointSettingsSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "Default endpoint settings for the project.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"pg_settings": schema.MapAttribute{
				Description: "A raw representation of Postgres settings.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"autoscaling_limit_min_cu": schema.Float64Attribute{
				Description: "The minimum number of Compute Units.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"autoscaling_limit_max_cu": schema.Float64Attribute{
				Description: "The maximum number of Compute Units.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"suspend_timeout_seconds": schema.Int64Attribute{
				Description: "Duration of inactivity in seconds after which the compute endpoint is automatically suspended.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func projectSettingsSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "Project settings.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: projectSettingsSchemaAttributes(),
	}
}

func projectSettingsSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"quota":                      projectQuotaSchema(),
		"allowed_ips":                projectAllowedIpsSchema(),
		"enable_logical_replication": projectEnableLogicalReplicationSchema(),
		"maintenance_window":         projectMaintenanceWindowSchema(),
		"block_public_connections":   projectBlockPublicConnectionsSchema(),
		"block_vpc_connections":      projectBlockVpcConnectionsSchema(),
		"audit_log_level":            projectAuditLogLevelSchema(),
		"hipaa":                      projectHipaaSchema(),
		"preload_libraries":          projectPreloadLibrariesSchema(),
	}
}

func projectQuotaSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "Per-project consumption quota.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"active_time_seconds": schema.Int64Attribute{
				Description: "The total amount of wall-clock time allowed to be spent by the project's compute endpoints.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"compute_time_seconds": schema.Int64Attribute{
				Description: "The total amount of CPU seconds allowed to be spent by the project's compute endpoints.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"written_data_bytes": schema.Int64Attribute{
				Description: "Total amount of data written to all of a project's branches.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"data_transfer_bytes": schema.Int64Attribute{
				Description: "Total amount of data transferred from all of a project's branches using the proxy.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"logical_size_bytes": schema.Int64Attribute{
				Description: "Limit on the logical size of every project's branch.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func projectAllowedIpsSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "A list of IP addresses that are allowed to connect to the endpoint.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"ips": schema.ListAttribute{
				Description: "A list of allowed IP addresses.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"protected_branches_only": schema.BoolAttribute{
				Description: "If true, the list will be applied only to protected branches.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func projectEnableLogicalReplicationSchema() schema.BoolAttribute {
	return schema.BoolAttribute{
		Description: "Sets wal_level=logical for all compute endpoints in this project.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	}
}

func projectMaintenanceWindowSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "The maintenance window configuration.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			// These three attributes must be Optional+Computed, not Required:
			// Required children inside an Optional+Computed parent that uses
			// UseStateForUnknown cause a perpetual post-apply diff whenever the
			// server populates maintenance_window but the practitioner never
			// configured it (Terraform core takes Required children from
			// config, i.e. null, when building the proposed new state, which
			// then never matches the UseStateForUnknown-restored prior state).
			// AlsoRequires below preserves all-or-nothing configuration.
			"weekdays": schema.ListAttribute{
				Description: "A list of weekdays when the maintenance window is active (1=Monday, 7=Sunday). Required together with start_time and end_time.",
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.List{
					listvalidator.AlsoRequires(
						path.MatchRelative().AtParent().AtName("start_time"),
						path.MatchRelative().AtParent().AtName("end_time"),
					),
				},
			},
			"start_time": schema.StringAttribute{
				Description: "Start time of the maintenance window in HH:MM format (UTC). Required together with weekdays and end_time.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.AlsoRequires(
						path.MatchRelative().AtParent().AtName("weekdays"),
						path.MatchRelative().AtParent().AtName("end_time"),
					),
				},
			},
			"end_time": schema.StringAttribute{
				Description: "End time of the maintenance window in HH:MM format (UTC). Required together with weekdays and start_time.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.AlsoRequires(
						path.MatchRelative().AtParent().AtName("weekdays"),
						path.MatchRelative().AtParent().AtName("start_time"),
					),
				},
			},
		},
	}
}

func projectBlockPublicConnectionsSchema() schema.BoolAttribute {
	return schema.BoolAttribute{
		Description: "When set, connections from the public internet are disallowed.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	}
}

func projectBlockVpcConnectionsSchema() schema.BoolAttribute {
	return schema.BoolAttribute{
		Description: "When set, connections using VPC endpoints are disallowed.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	}
}

func projectAuditLogLevelSchema() schema.StringAttribute {
	return schema.StringAttribute{
		Description: "The audit log level. One of: base, extended, full.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

func projectHipaaSchema() schema.BoolAttribute {
	return schema.BoolAttribute{
		Description: "Whether HIPAA compliance is enabled for the project.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	}
}

func projectPreloadLibrariesSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "Configuration for preloaded Postgres libraries.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"use_defaults": schema.BoolAttribute{
				Description: "Whether to use the default preloaded libraries.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled_libraries": schema.ListAttribute{
				Description: "A list of libraries to preload.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
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

	var config projectResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := buildProjectCreateFields(&data)

	if !data.DefaultEndpointSettings.IsNull() && !data.DefaultEndpointSettings.IsUnknown() &&
		!config.DefaultEndpointSettings.IsNull() && !config.DefaultEndpointSettings.IsUnknown() {
		des, included := buildDefaultEndpointSettingsRequest(ctx, data.DefaultEndpointSettings, config.DefaultEndpointSettings, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if included {
			project.DefaultEndpointSettings = neon.NewOptDefaultEndpointSettings(des)
		}
	}

	if !data.Settings.IsNull() && !data.Settings.IsUnknown() &&
		!config.Settings.IsNull() && !config.Settings.IsUnknown() {
		settings, included := buildProjectSettingsRequest(ctx, data.Settings, config.Settings, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if included {
			project.Settings = neon.NewOptProjectSettingsData(settings)
		}
	}

	apiReq := &neon.ProjectCreateRequest{Project: project}
	result, err := r.client.CreateProject(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create project", neonerror.Detail(err))
		return
	}

	mapProjectToModel(ctx, &result.Project, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func buildProjectCreateFields(data *projectResourceModel) neon.ProjectCreateRequestProject {
	p := neon.ProjectCreateRequestProject{}
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		p.Name = neon.NewOptString(data.Name.ValueString())
	}
	if !data.RegionID.IsNull() && !data.RegionID.IsUnknown() {
		p.RegionID = neon.NewOptString(data.RegionID.ValueString())
	}
	if !data.PgVersion.IsNull() && !data.PgVersion.IsUnknown() {
		p.PgVersion = neon.NewOptPgVersion(neon.PgVersion(data.PgVersion.ValueInt32()))
	}
	if !data.HistoryRetentionSeconds.IsNull() && !data.HistoryRetentionSeconds.IsUnknown() {
		p.HistoryRetentionSeconds = neon.NewOptInt32(data.HistoryRetentionSeconds.ValueInt32())
	}
	if !data.StorePasswords.IsNull() && !data.StorePasswords.IsUnknown() {
		p.StorePasswords = neon.NewOptBool(data.StorePasswords.ValueBool())
	}
	if !data.OrgID.IsNull() && !data.OrgID.IsUnknown() {
		p.OrgID = neon.NewOptString(data.OrgID.ValueString())
	}
	if !data.Provisioner.IsNull() && !data.Provisioner.IsUnknown() {
		p.Provisioner = neon.NewOptProvisioner(neon.Provisioner(data.Provisioner.ValueString()))
	}
	return p
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
		resp.Diagnostics.AddError("Failed to read project", neonerror.Detail(err))
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

	// The plan carries the resolved value for every attribute (including ones the
	// server computes), so it cannot tell us whether an attribute was actually
	// configured by the practitioner. Only the config can: an unconfigured
	// Optional+Computed attribute is null in config even though it is known in
	// plan. Gate what we send in the PATCH on config, but take the value to send
	// from the plan.
	var config projectResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &neon.ProjectUpdateRequest{
		Project: buildProjectUpdateFields(ctx, &data, &config, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.UpdateProject(ctx, apiReq, neon.UpdateProjectParams{
		ProjectID: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update project", neonerror.Detail(err))
		return
	}

	mapProjectToModel(ctx, &result.Project, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// buildProjectUpdateFields builds the PATCH request body for Update, gating each
// field on the corresponding config value being known and non-null (see the
// comment in Update) and taking the value to send from plan.
func buildProjectUpdateFields(ctx context.Context, plan, config *projectResourceModel, diags *diag.Diagnostics) neon.ProjectUpdateRequestProject {
	p := neon.ProjectUpdateRequestProject{}

	if !plan.Name.IsNull() && !plan.Name.IsUnknown() &&
		!config.Name.IsNull() && !config.Name.IsUnknown() {
		p.Name = neon.NewOptString(plan.Name.ValueString())
	}
	if !plan.HistoryRetentionSeconds.IsNull() && !plan.HistoryRetentionSeconds.IsUnknown() &&
		!config.HistoryRetentionSeconds.IsNull() && !config.HistoryRetentionSeconds.IsUnknown() {
		p.HistoryRetentionSeconds = neon.NewOptInt32(plan.HistoryRetentionSeconds.ValueInt32())
	}

	if !plan.DefaultEndpointSettings.IsNull() && !plan.DefaultEndpointSettings.IsUnknown() &&
		!config.DefaultEndpointSettings.IsNull() && !config.DefaultEndpointSettings.IsUnknown() {
		des, included := buildDefaultEndpointSettingsRequest(ctx, plan.DefaultEndpointSettings, config.DefaultEndpointSettings, diags)
		if included {
			p.DefaultEndpointSettings = neon.NewOptDefaultEndpointSettings(des)
		}
	}

	if !plan.Settings.IsNull() && !plan.Settings.IsUnknown() &&
		!config.Settings.IsNull() && !config.Settings.IsUnknown() {
		settings, included := buildProjectSettingsRequest(ctx, plan.Settings, config.Settings, diags)
		if included {
			p.Settings = neon.NewOptProjectSettingsData(settings)
		}
	}

	return p
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
		if neonerror.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Failed to delete project", neonerror.Detail(err))
		return
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// decodeObjectIfKnown decodes obj into T when it is known and non-null. When obj is
// null or unknown, decoding is skipped and the zero value of T is returned, whose
// tfsdk fields are all null (the zero value of a types.* value is its null state).
// This lets callers treat "not configured" and "explicitly configured" uniformly
// without ever calling ObjectValue.As on a null/unknown object, which strict typed
// decoding rejects.
func decodeObjectIfKnown[T any](ctx context.Context, obj basetypes.ObjectValue, diags *diag.Diagnostics) T {
	var m T
	if obj.IsNull() || obj.IsUnknown() {
		return m
	}
	diags.Append(obj.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	return m
}

// buildDefaultEndpointSettingsRequest builds the API request payload for
// default_endpoint_settings. Each leaf is included only when the corresponding
// leaf in cfg (the practitioner's configuration) is known and non-null; the value
// sent is always taken from plan. This ensures server-sourced values that merely
// flow through the plan (because the attribute is Optional+Computed) are never
// echoed back to the API. The returned bool reports whether any leaf was included;
// callers must not attach the object to the request when it is false.
func buildDefaultEndpointSettingsRequest(ctx context.Context, plan, cfg basetypes.ObjectValue, diags *diag.Diagnostics) (neon.DefaultEndpointSettings, bool) {
	pm := decodeObjectIfKnown[defaultEndpointSettingsModel](ctx, plan, diags)
	cm := decodeObjectIfKnown[defaultEndpointSettingsModel](ctx, cfg, diags)
	if diags.HasError() {
		return neon.DefaultEndpointSettings{}, false
	}

	des := neon.DefaultEndpointSettings{}
	included := false
	if !cm.AutoscalingLimitMinCu.IsNull() && !cm.AutoscalingLimitMinCu.IsUnknown() {
		des.AutoscalingLimitMinCu = neon.NewOptComputeUnit(neon.ComputeUnit(pm.AutoscalingLimitMinCu.ValueFloat64()))
		included = true
	}
	if !cm.AutoscalingLimitMaxCu.IsNull() && !cm.AutoscalingLimitMaxCu.IsUnknown() {
		des.AutoscalingLimitMaxCu = neon.NewOptComputeUnit(neon.ComputeUnit(pm.AutoscalingLimitMaxCu.ValueFloat64()))
		included = true
	}
	if !cm.SuspendTimeoutSeconds.IsNull() && !cm.SuspendTimeoutSeconds.IsUnknown() {
		des.SuspendTimeoutSeconds = neon.NewOptSuspendTimeoutSeconds(neon.SuspendTimeoutSeconds(pm.SuspendTimeoutSeconds.ValueInt64()))
		included = true
	}
	if !cm.PgSettings.IsNull() && !cm.PgSettings.IsUnknown() {
		pgSettings := make(map[string]string)
		diags.Append(pm.PgSettings.ElementsAs(ctx, &pgSettings, false)...)
		if diags.HasError() {
			return neon.DefaultEndpointSettings{}, false
		}
		des.PgSettings = neon.NewOptPgSettingsData(neon.PgSettingsData(pgSettings))
		included = true
	}
	return des, included
}

// buildProjectSettingsRequest builds the API request payload for settings, gating
// every leaf (including leaves inside nested objects) on the corresponding leaf in
// cfg being known and non-null, and taking values from plan. See
// buildDefaultEndpointSettingsRequest for the rationale. The returned bool reports
// whether any leaf was included.
func buildProjectSettingsRequest(ctx context.Context, plan, cfg basetypes.ObjectValue, diags *diag.Diagnostics) (neon.ProjectSettingsData, bool) {
	pm := decodeObjectIfKnown[projectSettingsModel](ctx, plan, diags)
	cm := decodeObjectIfKnown[projectSettingsModel](ctx, cfg, diags)
	if diags.HasError() {
		return neon.ProjectSettingsData{}, false
	}

	settings := neon.ProjectSettingsData{}
	included := buildProjectSettingsBoolFields(&pm, &cm, &settings)
	included = buildProjectQuotaRequest(ctx, &pm, &cm, &settings, diags) || included
	included = buildProjectAllowedIpsRequest(ctx, &pm, &cm, &settings, diags) || included
	included = buildProjectMaintenanceWindowRequest(ctx, &pm, &cm, &settings, diags) || included
	included = buildProjectPreloadLibrariesRequest(ctx, &pm, &cm, &settings, diags) || included
	if diags.HasError() {
		return neon.ProjectSettingsData{}, false
	}

	return settings, included
}

func buildProjectSettingsBoolFields(pm, cm *projectSettingsModel, settings *neon.ProjectSettingsData) bool {
	included := false
	if !cm.EnableLogicalReplication.IsNull() && !cm.EnableLogicalReplication.IsUnknown() {
		settings.EnableLogicalReplication = neon.NewOptBool(pm.EnableLogicalReplication.ValueBool())
		included = true
	}
	if !cm.BlockPublicConnections.IsNull() && !cm.BlockPublicConnections.IsUnknown() {
		settings.BlockPublicConnections = neon.NewOptBool(pm.BlockPublicConnections.ValueBool())
		included = true
	}
	if !cm.BlockVpcConnections.IsNull() && !cm.BlockVpcConnections.IsUnknown() {
		settings.BlockVpcConnections = neon.NewOptBool(pm.BlockVpcConnections.ValueBool())
		included = true
	}
	if !cm.AuditLogLevel.IsNull() && !cm.AuditLogLevel.IsUnknown() {
		settings.AuditLogLevel = neon.NewOptProjectAuditLogLevel(neon.ProjectAuditLogLevel(pm.AuditLogLevel.ValueString()))
		included = true
	}
	if !cm.Hipaa.IsNull() && !cm.Hipaa.IsUnknown() {
		settings.Hipaa = neon.NewOptBool(pm.Hipaa.ValueBool())
		included = true
	}
	return included
}

func buildProjectQuotaRequest(ctx context.Context, pm, cm *projectSettingsModel, settings *neon.ProjectSettingsData, diags *diag.Diagnostics) bool {
	if cm.Quota.IsNull() || cm.Quota.IsUnknown() {
		return false
	}
	cqm := decodeObjectIfKnown[projectQuotaModel](ctx, cm.Quota, diags)
	pqm := decodeObjectIfKnown[projectQuotaModel](ctx, pm.Quota, diags)
	if diags.HasError() {
		return false
	}

	quota := neon.ProjectQuota{}
	included := false
	if !cqm.ActiveTimeSeconds.IsNull() && !cqm.ActiveTimeSeconds.IsUnknown() {
		quota.ActiveTimeSeconds = neon.NewOptInt64(pqm.ActiveTimeSeconds.ValueInt64())
		included = true
	}
	if !cqm.ComputeTimeSeconds.IsNull() && !cqm.ComputeTimeSeconds.IsUnknown() {
		quota.ComputeTimeSeconds = neon.NewOptInt64(pqm.ComputeTimeSeconds.ValueInt64())
		included = true
	}
	if !cqm.WrittenDataBytes.IsNull() && !cqm.WrittenDataBytes.IsUnknown() {
		quota.WrittenDataBytes = neon.NewOptInt64(pqm.WrittenDataBytes.ValueInt64())
		included = true
	}
	if !cqm.DataTransferBytes.IsNull() && !cqm.DataTransferBytes.IsUnknown() {
		quota.DataTransferBytes = neon.NewOptInt64(pqm.DataTransferBytes.ValueInt64())
		included = true
	}
	if !cqm.LogicalSizeBytes.IsNull() && !cqm.LogicalSizeBytes.IsUnknown() {
		quota.LogicalSizeBytes = neon.NewOptInt64(pqm.LogicalSizeBytes.ValueInt64())
		included = true
	}
	if !included {
		// Empty nested objects are dangerous to send (see allowed_ips), and
		// pointless here since nothing was configured; skip attaching.
		return false
	}
	settings.Quota = neon.NewOptProjectQuota(quota)
	return true
}

func buildProjectAllowedIpsRequest(ctx context.Context, pm, cm *projectSettingsModel, settings *neon.ProjectSettingsData, diags *diag.Diagnostics) bool {
	if cm.AllowedIps.IsNull() || cm.AllowedIps.IsUnknown() {
		return false
	}
	caim := decodeObjectIfKnown[allowedIpsModel](ctx, cm.AllowedIps, diags)
	paim := decodeObjectIfKnown[allowedIpsModel](ctx, pm.AllowedIps, diags)
	if diags.HasError() {
		return false
	}

	allowedIps := neon.AllowedIps{}
	included := false
	if !caim.Ips.IsNull() && !caim.Ips.IsUnknown() {
		var ips []string
		diags.Append(paim.Ips.ElementsAs(ctx, &ips, false)...)
		if diags.HasError() {
			return false
		}
		allowedIps.Ips = ips
		included = true
	}
	if !caim.ProtectedBranchesOnly.IsNull() && !caim.ProtectedBranchesOnly.IsUnknown() {
		allowedIps.ProtectedBranchesOnly = neon.NewOptBool(paim.ProtectedBranchesOnly.ValueBool())
		included = true
	}
	if !included {
		// An empty allowed_ips object can mean "allow all IPs" server-side;
		// never send it unless at least one leaf was actually configured.
		return false
	}
	settings.AllowedIps = neon.NewOptAllowedIps(allowedIps)
	return true
}

func buildProjectMaintenanceWindowRequest(ctx context.Context, pm, cm *projectSettingsModel, settings *neon.ProjectSettingsData, diags *diag.Diagnostics) bool {
	// maintenance_window's children are Optional+Computed (not Required; see
	// the schema comment for why), but the AlsoRequires validators on each
	// child guarantee all-or-nothing configuration: if the object is
	// configured in config at all, every child is necessarily known and
	// non-null there too. We still defensively check the plan-side values
	// before reading them, since they're a separate value from cfg.
	if cm.MaintenanceWindow.IsNull() || cm.MaintenanceWindow.IsUnknown() {
		return false
	}
	if pm.MaintenanceWindow.IsNull() || pm.MaintenanceWindow.IsUnknown() {
		return false
	}
	var mwm maintenanceWindowModel
	diags.Append(pm.MaintenanceWindow.As(ctx, &mwm, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return false
	}
	if mwm.StartTime.IsNull() || mwm.StartTime.IsUnknown() ||
		mwm.EndTime.IsNull() || mwm.EndTime.IsUnknown() ||
		mwm.Weekdays.IsNull() || mwm.Weekdays.IsUnknown() {
		return false
	}
	mw := neon.MaintenanceWindow{
		StartTime: mwm.StartTime.ValueString(),
		EndTime:   mwm.EndTime.ValueString(),
	}
	var weekdays []int
	diags.Append(mwm.Weekdays.ElementsAs(ctx, &weekdays, false)...)
	if diags.HasError() {
		return false
	}
	mw.Weekdays = weekdays
	settings.MaintenanceWindow = neon.NewOptMaintenanceWindow(mw)
	return true
}

func buildProjectPreloadLibrariesRequest(ctx context.Context, pm, cm *projectSettingsModel, settings *neon.ProjectSettingsData, diags *diag.Diagnostics) bool {
	if cm.PreloadLibraries.IsNull() || cm.PreloadLibraries.IsUnknown() {
		return false
	}
	cplm := decodeObjectIfKnown[preloadLibrariesModel](ctx, cm.PreloadLibraries, diags)
	pplm := decodeObjectIfKnown[preloadLibrariesModel](ctx, pm.PreloadLibraries, diags)
	if diags.HasError() {
		return false
	}

	pl := neon.PreloadLibraries{}
	included := false
	if !cplm.UseDefaults.IsNull() && !cplm.UseDefaults.IsUnknown() {
		pl.UseDefaults = neon.NewOptBool(pplm.UseDefaults.ValueBool())
		included = true
	}
	if !cplm.EnabledLibraries.IsNull() && !cplm.EnabledLibraries.IsUnknown() {
		var libs []string
		diags.Append(pplm.EnabledLibraries.ElementsAs(ctx, &libs, false)...)
		if diags.HasError() {
			return false
		}
		pl.EnabledLibraries = libs
		included = true
	}
	if !included {
		return false
	}
	settings.PreloadLibraries = neon.NewOptPreloadLibraries(pl)
	return true
}

func mapProjectToModel(ctx context.Context, p *neon.Project, data *projectResourceModel, diags *diag.Diagnostics) {
	mapProjectCoreFields(p, data)
	mapProjectDefaultEndpointSettings(ctx, p, data, diags)
	mapProjectSettings(ctx, p, data, diags)
}

func mapProjectCoreFields(p *neon.Project, data *projectResourceModel) {
	data.ID = types.StringValue(p.ID)
	data.Name = types.StringValue(p.Name)
	data.RegionID = types.StringValue(p.RegionID)
	data.PgVersion = types.Int32Value(int32(p.PgVersion)) //nolint:gosec // PgVersion is a small Postgres version number, no overflow risk
	data.HistoryRetentionSeconds = types.Int32Value(p.HistoryRetentionSeconds)
	data.StorePasswords = types.BoolValue(p.StorePasswords)
	data.Provisioner = types.StringValue(string(p.Provisioner))
	data.CreatedAt = types.StringValue(p.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(p.UpdatedAt.Format(time.RFC3339))

	if p.OrgID.IsSet() {
		data.OrgID = types.StringValue(p.OrgID.Value)
	} else {
		data.OrgID = types.StringNull()
	}
}

func mapProjectDefaultEndpointSettings(ctx context.Context, p *neon.Project, data *projectResourceModel, diags *diag.Diagnostics) {
	if !p.DefaultEndpointSettings.IsSet() {
		data.DefaultEndpointSettings = types.ObjectNull(defaultEndpointSettingsAttrTypes)
		return
	}

	des := p.DefaultEndpointSettings.Value
	m := defaultEndpointSettingsModel{
		AutoscalingLimitMinCu: types.Float64Null(),
		AutoscalingLimitMaxCu: types.Float64Null(),
		SuspendTimeoutSeconds: types.Int64Null(),
		PgSettings:            types.MapNull(types.StringType),
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
	obj, d := types.ObjectValueFrom(ctx, defaultEndpointSettingsAttrTypes, m)
	diags.Append(d...)
	data.DefaultEndpointSettings = obj
}

func mapProjectSettings(ctx context.Context, p *neon.Project, data *projectResourceModel, diags *diag.Diagnostics) {
	if !p.Settings.IsSet() {
		data.Settings = types.ObjectNull(settingsAttrTypes)
		return
	}

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

	mapProjectSettingsBoolFields(&s, &m)
	mapProjectQuotaToModel(ctx, &s, &m, diags)
	mapProjectAllowedIpsToModel(ctx, &s, &m, diags)
	mapProjectMaintenanceWindowToModel(ctx, &s, &m, diags)
	mapProjectPreloadLibrariesToModel(ctx, &s, &m, diags)

	obj, d := types.ObjectValueFrom(ctx, settingsAttrTypes, m)
	diags.Append(d...)
	data.Settings = obj
}

func mapProjectSettingsBoolFields(s *neon.ProjectSettingsData, m *projectSettingsModel) {
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
}

func mapProjectQuotaToModel(ctx context.Context, s *neon.ProjectSettingsData, m *projectSettingsModel, diags *diag.Diagnostics) {
	if !s.Quota.IsSet() {
		return
	}
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

func mapProjectAllowedIpsToModel(ctx context.Context, s *neon.ProjectSettingsData, m *projectSettingsModel, diags *diag.Diagnostics) {
	if !s.AllowedIps.IsSet() {
		return
	}
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

func mapProjectMaintenanceWindowToModel(ctx context.Context, s *neon.ProjectSettingsData, m *projectSettingsModel, diags *diag.Diagnostics) {
	if !s.MaintenanceWindow.IsSet() {
		return
	}
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

func mapProjectPreloadLibrariesToModel(ctx context.Context, s *neon.ProjectSettingsData, m *projectSettingsModel, diags *diag.Diagnostics) {
	if !s.PreloadLibraries.IsSet() {
		return
	}
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
