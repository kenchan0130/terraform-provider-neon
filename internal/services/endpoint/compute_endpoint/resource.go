package compute_endpoint

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonerror"
)

var (
	_ resource.Resource                = &endpointResource{}
	_ resource.ResourceWithConfigure   = &endpointResource{}
	_ resource.ResourceWithImportState = &endpointResource{}
)

type endpointResource struct {
	client *neon.Client
}

var preloadLibrariesAttrTypes = map[string]attr.Type{
	"use_defaults":      types.BoolType,
	"enabled_libraries": types.ListType{ElemType: types.StringType},
}

var settingsAttrTypes = map[string]attr.Type{
	"pg_settings":        types.MapType{ElemType: types.StringType},
	"pgbouncer_settings": types.MapType{ElemType: types.StringType},
	"preload_libraries":  types.ObjectType{AttrTypes: preloadLibrariesAttrTypes},
}

type endpointResourceModel struct {
	ID                    types.String  `tfsdk:"id"`
	ProjectID             types.String  `tfsdk:"project_id"`
	BranchID              types.String  `tfsdk:"branch_id"`
	Type                  types.String  `tfsdk:"type"`
	Name                  types.String  `tfsdk:"name"`
	AutoscalingLimitMinCu types.Float64 `tfsdk:"autoscaling_limit_min_cu"`
	AutoscalingLimitMaxCu types.Float64 `tfsdk:"autoscaling_limit_max_cu"`
	SuspendTimeoutSeconds types.Int64   `tfsdk:"suspend_timeout_seconds"`
	PoolerEnabled         types.Bool    `tfsdk:"pooler_enabled"`
	PoolerMode            types.String  `tfsdk:"pooler_mode"`
	Disabled              types.Bool    `tfsdk:"disabled"`
	PasswordlessAccess    types.Bool    `tfsdk:"passwordless_access"`
	Provisioner           types.String  `tfsdk:"compute_provisioner"`
	Settings              types.Object  `tfsdk:"settings"`
	Host                  types.String  `tfsdk:"host"`
	RegionID              types.String  `tfsdk:"region_id"`
	CurrentState          types.String  `tfsdk:"current_state"`
	LastActive            types.String  `tfsdk:"last_active"`
	CreationSource        types.String  `tfsdk:"creation_source"`
	ComputeReleaseVersion types.String  `tfsdk:"compute_release_version"`
	PendingState          types.String  `tfsdk:"pending_state"`
	StartedAt             types.String  `tfsdk:"started_at"`
	SuspendedAt           types.String  `tfsdk:"suspended_at"`
	CreatedAt             types.String  `tfsdk:"created_at"`
	UpdatedAt             types.String  `tfsdk:"updated_at"`
}

func NewResource() resource.Resource {
	return &endpointResource{}
}

func (r *endpointResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_endpoint"
}

func (r *endpointResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Neon endpoint (compute).",
		Attributes:  endpointResourceSchemaAttributes(),
	}
}

func endpointResourceSchemaConfigurableAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "The endpoint ID.",
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
		"branch_id": schema.StringAttribute{
			Description: "The branch ID.",
			Required:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"type": schema.StringAttribute{
			Description: "The endpoint type. Must be `read_write` or `read_only`.",
			Required:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"name": schema.StringAttribute{
			Description: "Optional name of the compute endpoint.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
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
			Description: "The duration of inactivity in seconds after which the compute is suspended.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"pooler_enabled": schema.BoolAttribute{
			Description: "Whether connection pooling is enabled.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"pooler_mode": schema.StringAttribute{
			Description: "The connection pooler mode. Must be `transaction`.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"disabled": schema.BoolAttribute{
			Description: "Whether the endpoint is disabled.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"passwordless_access": schema.BoolAttribute{
			Description: "Whether to permit passwordless access to the compute endpoint.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"compute_provisioner": schema.StringAttribute{
			Description: "The provisioner for the compute endpoint.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"settings": endpointSettingsResourceSchema(),
		"region_id": schema.StringAttribute{
			Description: "The region identifier.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func endpointSettingsResourceSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "Endpoint settings.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"pg_settings": schema.MapAttribute{
				Description: "A raw representation of Postgres settings.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"pgbouncer_settings": schema.MapAttribute{
				Description: "A raw representation of PgBouncer settings.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"preload_libraries": schema.SingleNestedAttribute{
				Description: "Preload libraries configuration.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"use_defaults": schema.BoolAttribute{
						Description: "Whether to use default preload libraries.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"enabled_libraries": schema.ListAttribute{
						Description: "List of enabled preload libraries.",
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

func endpointSchemaComputedAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"host": schema.StringAttribute{
			Description: "The hostname for connecting to the endpoint.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"current_state": schema.StringAttribute{
			Description: "The current state of the compute endpoint.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"last_active": schema.StringAttribute{
			Description: "A timestamp indicating when the compute endpoint was last active.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"creation_source": schema.StringAttribute{
			Description: "The compute endpoint creation source.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"compute_release_version": schema.StringAttribute{
			Description: "Attached compute's release version number.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"pending_state": schema.StringAttribute{
			Description: "The pending state of the compute endpoint.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"started_at": schema.StringAttribute{
			Description: "A timestamp indicating when the compute endpoint was last started.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"suspended_at": schema.StringAttribute{
			Description: "A timestamp indicating when the compute endpoint was last suspended.",
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
	}
}

func endpointResourceSchemaAttributes() map[string]schema.Attribute {
	attrs := endpointResourceSchemaConfigurableAttributes()
	for k, v := range endpointSchemaComputedAttributes() {
		attrs[k] = v
	}
	return attrs
}

func (r *endpointResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *endpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data endpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ep := neon.EndpointCreateRequestEndpoint{
		BranchID: data.BranchID.ValueString(),
		Type:     neon.EndpointType(data.Type.ValueString()),
	}

	if !data.RegionID.IsNull() && !data.RegionID.IsUnknown() {
		ep.RegionID = neon.NewOptString(data.RegionID.ValueString())
	}

	setEndpointCommonFields(&data, &ep.Name, &ep.AutoscalingLimitMinCu, &ep.AutoscalingLimitMaxCu,
		&ep.SuspendTimeoutSeconds, &ep.PoolerEnabled, &ep.PoolerMode, &ep.Disabled, //nolint:staticcheck // intentionally using deprecated API field for backward compatibility
		&ep.PasswordlessAccess, &ep.Provisioner)

	buildSettingsRequest(ctx, data.Settings, &ep.Settings, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &neon.EndpointCreateRequest{Endpoint: ep}

	result, err := r.client.CreateProjectEndpoint(ctx, apiReq, neon.CreateProjectEndpointParams{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create endpoint", err.Error())
		return
	}

	mapEndpointToModel(ctx, &result.Endpoint, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *endpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data endpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetProjectEndpoint(ctx, neon.GetProjectEndpointParams{
		ProjectID:  data.ProjectID.ValueString(),
		EndpointID: data.ID.ValueString(),
	})
	if err != nil {
		if neonerror.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read endpoint", err.Error())
		return
	}

	mapEndpointToModel(ctx, &result.Endpoint, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *endpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data endpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state endpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ep := neon.EndpointUpdateRequestEndpoint{}

	setEndpointCommonFields(&data, &ep.Name, &ep.AutoscalingLimitMinCu, &ep.AutoscalingLimitMaxCu,
		&ep.SuspendTimeoutSeconds, &ep.PoolerEnabled, &ep.PoolerMode, &ep.Disabled, //nolint:staticcheck // intentionally using deprecated API field for backward compatibility
		&ep.PasswordlessAccess, &ep.Provisioner)

	buildSettingsRequest(ctx, data.Settings, &ep.Settings, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &neon.EndpointUpdateRequest{Endpoint: ep}

	result, err := r.client.UpdateProjectEndpoint(ctx, apiReq, neon.UpdateProjectEndpointParams{
		ProjectID:  state.ProjectID.ValueString(),
		EndpointID: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update endpoint", err.Error())
		return
	}

	mapEndpointToModel(ctx, &result.Endpoint, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *endpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data endpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectEndpoint(ctx, neon.DeleteProjectEndpointParams{
		ProjectID:  data.ProjectID.ValueString(),
		EndpointID: data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete endpoint", err.Error())
		return
	}
}

func (r *endpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: {project_id}/{endpoint_id}",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func setEndpointCommonFields(
	data *endpointResourceModel,
	name *neon.OptString,
	minCu *neon.OptComputeUnit,
	maxCu *neon.OptComputeUnit,
	suspendTimeout *neon.OptSuspendTimeoutSeconds,
	poolerEnabled *neon.OptBool,
	poolerMode *neon.OptEndpointPoolerMode,
	disabled *neon.OptBool,
	passwordlessAccess *neon.OptBool,
	provisioner *neon.OptProvisioner,
) {
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		*name = neon.NewOptString(data.Name.ValueString())
	}
	if !data.AutoscalingLimitMinCu.IsNull() && !data.AutoscalingLimitMinCu.IsUnknown() {
		*minCu = neon.NewOptComputeUnit(neon.ComputeUnit(data.AutoscalingLimitMinCu.ValueFloat64()))
	}
	if !data.AutoscalingLimitMaxCu.IsNull() && !data.AutoscalingLimitMaxCu.IsUnknown() {
		*maxCu = neon.NewOptComputeUnit(neon.ComputeUnit(data.AutoscalingLimitMaxCu.ValueFloat64()))
	}
	if !data.SuspendTimeoutSeconds.IsNull() && !data.SuspendTimeoutSeconds.IsUnknown() {
		*suspendTimeout = neon.NewOptSuspendTimeoutSeconds(neon.SuspendTimeoutSeconds(data.SuspendTimeoutSeconds.ValueInt64()))
	}
	if !data.PoolerEnabled.IsNull() && !data.PoolerEnabled.IsUnknown() {
		*poolerEnabled = neon.NewOptBool(data.PoolerEnabled.ValueBool())
	}
	if !data.PoolerMode.IsNull() && !data.PoolerMode.IsUnknown() {
		*poolerMode = neon.NewOptEndpointPoolerMode(neon.EndpointPoolerMode(data.PoolerMode.ValueString()))
	}
	if !data.Disabled.IsNull() && !data.Disabled.IsUnknown() {
		*disabled = neon.NewOptBool(data.Disabled.ValueBool())
	}
	if !data.PasswordlessAccess.IsNull() && !data.PasswordlessAccess.IsUnknown() {
		*passwordlessAccess = neon.NewOptBool(data.PasswordlessAccess.ValueBool())
	}
	if !data.Provisioner.IsNull() && !data.Provisioner.IsUnknown() {
		*provisioner = neon.NewOptProvisioner(neon.Provisioner(data.Provisioner.ValueString()))
	}
}

func buildSettingsRequest(ctx context.Context, settings types.Object, target *neon.OptEndpointSettingsData, diags *diag.Diagnostics) {
	if settings.IsNull() || settings.IsUnknown() {
		return
	}

	attrs := settings.Attributes()
	s := neon.EndpointSettingsData{}

	buildPgSettingsRequest(ctx, attrs, &s.PgSettings, diags)
	if diags.HasError() {
		return
	}

	buildPgbouncerSettingsRequest(ctx, attrs, &s.PgbouncerSettings, diags)
	if diags.HasError() {
		return
	}

	buildPreloadLibrariesRequest(ctx, attrs, &s.PreloadLibraries, diags)
	if diags.HasError() {
		return
	}

	*target = neon.NewOptEndpointSettingsData(s)
}

func buildPgSettingsRequest(ctx context.Context, attrs map[string]attr.Value, target *neon.OptPgSettingsData, diags *diag.Diagnostics) {
	pgSettings, ok := attrs["pg_settings"]
	if !ok {
		return
	}
	pgMap, ok := pgSettings.(types.Map)
	if !ok || pgMap.IsNull() || pgMap.IsUnknown() {
		return
	}
	m := make(map[string]string)
	diags.Append(pgMap.ElementsAs(ctx, &m, false)...)
	if !diags.HasError() {
		*target = neon.NewOptPgSettingsData(neon.PgSettingsData(m))
	}
}

func buildPgbouncerSettingsRequest(ctx context.Context, attrs map[string]attr.Value, target *neon.OptPgbouncerSettingsData, diags *diag.Diagnostics) {
	pgbSettings, ok := attrs["pgbouncer_settings"]
	if !ok {
		return
	}
	pgbMap, ok := pgbSettings.(types.Map)
	if !ok || pgbMap.IsNull() || pgbMap.IsUnknown() {
		return
	}
	m := make(map[string]string)
	diags.Append(pgbMap.ElementsAs(ctx, &m, false)...)
	if !diags.HasError() {
		*target = neon.NewOptPgbouncerSettingsData(neon.PgbouncerSettingsData(m))
	}
}

func buildPreloadLibrariesRequest(ctx context.Context, attrs map[string]attr.Value, target *neon.OptPreloadLibraries, diags *diag.Diagnostics) {
	plAttr, ok := attrs["preload_libraries"]
	if !ok {
		return
	}
	plObj, ok := plAttr.(types.Object)
	if !ok || plObj.IsNull() || plObj.IsUnknown() {
		return
	}

	plAttrs := plObj.Attributes()
	pl := neon.PreloadLibraries{}

	if ud, ok := plAttrs["use_defaults"]; ok {
		udBool, ok := ud.(types.Bool)
		if ok && !udBool.IsNull() && !udBool.IsUnknown() {
			pl.UseDefaults = neon.NewOptBool(udBool.ValueBool())
		}
	}

	if el, ok := plAttrs["enabled_libraries"]; ok {
		elList, ok := el.(types.List)
		if ok && !elList.IsNull() && !elList.IsUnknown() {
			var libs []string
			diags.Append(elList.ElementsAs(ctx, &libs, false)...)
			if diags.HasError() {
				return
			}
			pl.EnabledLibraries = libs
		}
	}

	*target = neon.NewOptPreloadLibraries(pl)
}

func mapEndpointToModel(_ context.Context, ep *neon.Endpoint, data *endpointResourceModel, diags *diag.Diagnostics) {
	mapEndpointCoreFields(ep, data)
	mapEndpointOptionalFields(ep, data)
	mapEndpointSettingsToModel(ep, data, diags)
}

func mapEndpointCoreFields(ep *neon.Endpoint, data *endpointResourceModel) {
	data.ID = types.StringValue(ep.ID)
	data.ProjectID = types.StringValue(ep.ProjectID)
	data.BranchID = types.StringValue(ep.BranchID)
	data.Type = types.StringValue(string(ep.Type))
	data.AutoscalingLimitMinCu = types.Float64Value(float64(ep.AutoscalingLimitMinCu))
	data.AutoscalingLimitMaxCu = types.Float64Value(float64(ep.AutoscalingLimitMaxCu))
	data.SuspendTimeoutSeconds = types.Int64Value(int64(ep.SuspendTimeoutSeconds))
	data.PoolerEnabled = types.BoolValue(ep.PoolerEnabled)
	data.PoolerMode = types.StringValue(string(ep.PoolerMode))
	data.Disabled = types.BoolValue(ep.Disabled)
	data.PasswordlessAccess = types.BoolValue(ep.PasswordlessAccess)
	data.Host = types.StringValue(ep.Host)
	data.RegionID = types.StringValue(ep.RegionID)
	data.CurrentState = types.StringValue(string(ep.CurrentState))
	data.CreationSource = types.StringValue(ep.CreationSource)
	data.Provisioner = types.StringValue(string(ep.Provisioner))
	data.CreatedAt = types.StringValue(ep.CreatedAt.String())
	data.UpdatedAt = types.StringValue(ep.UpdatedAt.String())
}

func mapEndpointOptionalFields(ep *neon.Endpoint, data *endpointResourceModel) {
	if v, ok := ep.Name.Get(); ok {
		data.Name = types.StringValue(v)
	} else {
		data.Name = types.StringNull()
	}

	if ep.LastActive.IsSet() {
		data.LastActive = types.StringValue(ep.LastActive.Value.String())
	} else {
		data.LastActive = types.StringNull()
	}

	if v, ok := ep.ComputeReleaseVersion.Get(); ok {
		data.ComputeReleaseVersion = types.StringValue(v)
	} else {
		data.ComputeReleaseVersion = types.StringNull()
	}

	if ep.PendingState.IsSet() {
		data.PendingState = types.StringValue(string(ep.PendingState.Value))
	} else {
		data.PendingState = types.StringNull()
	}

	if ep.StartedAt.IsSet() {
		data.StartedAt = types.StringValue(ep.StartedAt.Value.String())
	} else {
		data.StartedAt = types.StringNull()
	}

	if ep.SuspendedAt.IsSet() {
		data.SuspendedAt = types.StringValue(ep.SuspendedAt.Value.String())
	} else {
		data.SuspendedAt = types.StringNull()
	}
}

func mapEndpointSettingsToModel(ep *neon.Endpoint, data *endpointResourceModel, diags *diag.Diagnostics) {
	settingsAttrs := map[string]attr.Value{}

	if ep.Settings.PgSettings.IsSet() {
		elems := make(map[string]attr.Value)
		for k, v := range ep.Settings.PgSettings.Value {
			elems[k] = types.StringValue(v)
		}
		pgMap, d := types.MapValue(types.StringType, elems)
		diags.Append(d...)
		settingsAttrs["pg_settings"] = pgMap
	} else {
		settingsAttrs["pg_settings"] = types.MapNull(types.StringType)
	}

	if ep.Settings.PgbouncerSettings.IsSet() {
		elems := make(map[string]attr.Value)
		for k, v := range ep.Settings.PgbouncerSettings.Value {
			elems[k] = types.StringValue(v)
		}
		pgbMap, d := types.MapValue(types.StringType, elems)
		diags.Append(d...)
		settingsAttrs["pgbouncer_settings"] = pgbMap
	} else {
		settingsAttrs["pgbouncer_settings"] = types.MapNull(types.StringType)
	}

	if ep.Settings.PreloadLibraries.IsSet() {
		plAttrs := map[string]attr.Value{}

		if ep.Settings.PreloadLibraries.Value.UseDefaults.IsSet() {
			plAttrs["use_defaults"] = types.BoolValue(ep.Settings.PreloadLibraries.Value.UseDefaults.Value)
		} else {
			plAttrs["use_defaults"] = types.BoolNull()
		}

		libElems := make([]attr.Value, len(ep.Settings.PreloadLibraries.Value.EnabledLibraries))
		for i, v := range ep.Settings.PreloadLibraries.Value.EnabledLibraries {
			libElems[i] = types.StringValue(v)
		}
		libList, d := types.ListValue(types.StringType, libElems)
		diags.Append(d...)
		plAttrs["enabled_libraries"] = libList

		plObj, d := types.ObjectValue(preloadLibrariesAttrTypes, plAttrs)
		diags.Append(d...)
		settingsAttrs["preload_libraries"] = plObj
	} else {
		settingsAttrs["preload_libraries"] = types.ObjectNull(preloadLibrariesAttrTypes)
	}

	settingsObj, d := types.ObjectValue(settingsAttrTypes, settingsAttrs)
	diags.Append(d...)
	data.Settings = settingsObj
}
