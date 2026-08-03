package project

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

func mustObjectValue(t *testing.T, attrTypes map[string]attr.Type, attrs map[string]attr.Value) basetypes.ObjectValue {
	t.Helper()
	obj, diags := types.ObjectValue(attrTypes, attrs)
	if diags.HasError() {
		t.Fatalf("failed to build object value: %v", diags)
	}
	return obj
}

func mustListValue(t *testing.T, elemType attr.Type, elems []attr.Value) basetypes.ListValue {
	t.Helper()
	list, diags := types.ListValue(elemType, elems)
	if diags.HasError() {
		t.Fatalf("failed to build list value: %v", diags)
	}
	return list
}

func mustSetValue(t *testing.T, elemType attr.Type, elems []attr.Value) basetypes.SetValue {
	t.Helper()
	set, diags := types.SetValue(elemType, elems)
	if diags.HasError() {
		t.Fatalf("failed to build set value: %v", diags)
	}
	return set
}

func mustMapValue(t *testing.T, elemType attr.Type, elems map[string]attr.Value) basetypes.MapValue {
	t.Helper()
	m, diags := types.MapValue(elemType, elems)
	if diags.HasError() {
		t.Fatalf("failed to build map value: %v", diags)
	}
	return m
}

// fullyPopulatedSettingsPlan builds a settings plan object where every leaf,
// including every leaf inside every nested object, is known and non-null. This
// is required to safely exercise the "cfg object is unknown" cascade (every leaf
// gets read straight from plan, including collections via ElementsAs, which
// errors on a null - not just unknown - collection).
func fullyPopulatedSettingsPlan(t *testing.T) basetypes.ObjectValue {
	t.Helper()

	quota := mustObjectValue(t, quotaAttrTypes, map[string]attr.Value{
		"active_time_seconds":  types.Int64Value(1),
		"compute_time_seconds": types.Int64Value(2),
		"written_data_bytes":   types.Int64Value(3),
		"data_transfer_bytes":  types.Int64Value(4),
		"logical_size_bytes":   types.Int64Value(5),
	})
	allowedIps := mustObjectValue(t, allowedIpsAttrTypes, map[string]attr.Value{
		"ips":                     mustListValue(t, types.StringType, []attr.Value{types.StringValue("0.0.0.0/0")}),
		"protected_branches_only": types.BoolValue(true),
	})
	maintenanceWindow := mustObjectValue(t, maintenanceWindowAttrTypes, map[string]attr.Value{
		"weekdays":   mustSetValue(t, types.Int64Type, []attr.Value{types.Int64Value(1)}),
		"start_time": types.StringValue("01:00"),
		"end_time":   types.StringValue("02:00"),
	})
	preloadLibraries := mustObjectValue(t, preloadLibrariesAttrTypes, map[string]attr.Value{
		"use_defaults":      types.BoolValue(true),
		"enabled_libraries": mustListValue(t, types.StringType, []attr.Value{types.StringValue("pg_stat_statements")}),
	})

	return mustObjectValue(t, settingsAttrTypes, map[string]attr.Value{
		"quota":                      quota,
		"allowed_ips":                allowedIps,
		"enable_logical_replication": types.BoolValue(true),
		"maintenance_window":         maintenanceWindow,
		"block_public_connections":   types.BoolValue(true),
		"block_vpc_connections":      types.BoolValue(true),
		"audit_log_level":            types.StringValue("full"),
		"hipaa":                      types.BoolValue(true),
		"preload_libraries":          preloadLibraries,
	})
}

// TestBuildDefaultEndpointSettingsRequest_UnconfiguredOrEmpty verifies that null,
// or empty-but-known config objects never produce diagnostics and never cause a
// request to be attached (no empty {} emitted). A null config object is the only
// thing that proves absence; see
// TestBuildDefaultEndpointSettingsRequest_UnknownConfigIncludedFromPlan for the
// unknown-config case, which behaves differently (included=true).
func TestBuildDefaultEndpointSettingsRequest_UnconfiguredOrEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := mustObjectValue(t, defaultEndpointSettingsAttrTypes, map[string]attr.Value{
		"pg_settings":              types.MapNull(types.StringType),
		"autoscaling_limit_min_cu": types.Float64Value(0.25),
		"autoscaling_limit_max_cu": types.Float64Value(0.25),
		"suspend_timeout_seconds":  types.Int64Value(300),
	})

	tests := map[string]struct {
		cfg basetypes.ObjectValue
	}{
		"null config": {
			cfg: types.ObjectNull(defaultEndpointSettingsAttrTypes),
		},
		"empty-but-known config (all leaves null)": {
			cfg: mustObjectValue(t, defaultEndpointSettingsAttrTypes, map[string]attr.Value{
				"pg_settings":              types.MapNull(types.StringType),
				"autoscaling_limit_min_cu": types.Float64Null(),
				"autoscaling_limit_max_cu": types.Float64Null(),
				"suspend_timeout_seconds":  types.Int64Null(),
			}),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var diags diag.Diagnostics
			des, included := buildDefaultEndpointSettingsRequest(ctx, plan, tt.cfg, &diags)
			if diags.HasError() {
				t.Errorf("unexpected diagnostics: %v", diags)
			}
			if included {
				t.Errorf("expected included=false, got true (des=%+v)", des)
			}
		})
	}
}

// TestBuildDefaultEndpointSettingsRequest_UnknownConfigIncludedFromPlan is the
// regression test for finding 1 (unknown config must NOT be treated as
// unconfigured): only a null config value proves the practitioner never set
// default_endpoint_settings. An unknown config value means an unresolved
// expression, not absence, so every leaf must be included, taken from plan.
func TestBuildDefaultEndpointSettingsRequest_UnknownConfigIncludedFromPlan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := mustObjectValue(t, defaultEndpointSettingsAttrTypes, map[string]attr.Value{
		"pg_settings":              mustMapValue(t, types.StringType, map[string]attr.Value{"foo": types.StringValue("bar")}),
		"autoscaling_limit_min_cu": types.Float64Value(0.25),
		"autoscaling_limit_max_cu": types.Float64Value(0.5),
		"suspend_timeout_seconds":  types.Int64Value(300),
	})
	cfg := types.ObjectUnknown(defaultEndpointSettingsAttrTypes)

	var diags diag.Diagnostics
	des, included := buildDefaultEndpointSettingsRequest(ctx, plan, cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !included {
		t.Fatal("expected included=true when config is unknown")
	}

	minCu, ok := des.AutoscalingLimitMinCu.Get()
	if !ok || float64(minCu) != 0.25 {
		t.Errorf("expected autoscaling_limit_min_cu=0.25 from plan, got %+v", des.AutoscalingLimitMinCu)
	}
	maxCu, ok := des.AutoscalingLimitMaxCu.Get()
	if !ok || float64(maxCu) != 0.5 {
		t.Errorf("expected autoscaling_limit_max_cu=0.5 from plan, got %+v", des.AutoscalingLimitMaxCu)
	}
	suspend, ok := des.SuspendTimeoutSeconds.Get()
	if !ok || int64(suspend) != 300 {
		t.Errorf("expected suspend_timeout_seconds=300 from plan, got %+v", des.SuspendTimeoutSeconds)
	}
	pg, ok := des.PgSettings.Get()
	if !ok || pg["foo"] != "bar" {
		t.Errorf("expected pg_settings={foo:bar} from plan, got %+v", des.PgSettings)
	}
}

// TestBuildProjectSettingsRequest_UnconfiguredOrEmpty covers the null-config and
// empty-but-known-config cases for the top-level settings object, and the EMPTY
// OBJECT BAN for each nested object (quota / allowed_ips / maintenance_window /
// preload_libraries): an unconfigured or empty nested object must never be
// attached, since e.g. an empty allowed_ips object can mean "allow all IPs"
// server-side, and an empty maintenance_window would otherwise recreate the
// forbidden-echo bug (finding 2). See
// TestBuildProjectSettingsRequest_UnknownConfigIncludedFromPlan for the
// unknown-config case, which behaves differently (included=true).
func TestBuildProjectSettingsRequest_UnconfiguredOrEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := fullyPopulatedSettingsPlan(t)

	tests := map[string]struct {
		cfg basetypes.ObjectValue
	}{
		"null config": {
			cfg: types.ObjectNull(settingsAttrTypes),
		},
		"empty-but-known config (all leaves null)": {
			cfg: mustObjectValue(t, settingsAttrTypes, map[string]attr.Value{
				"quota":                      types.ObjectNull(quotaAttrTypes),
				"allowed_ips":                types.ObjectNull(allowedIpsAttrTypes),
				"enable_logical_replication": types.BoolNull(),
				"maintenance_window":         types.ObjectNull(maintenanceWindowAttrTypes),
				"block_public_connections":   types.BoolNull(),
				"block_vpc_connections":      types.BoolNull(),
				"audit_log_level":            types.StringNull(),
				"hipaa":                      types.BoolNull(),
				"preload_libraries":          types.ObjectNull(preloadLibrariesAttrTypes),
			}),
		},
		"quota configured but empty (known, all leaves null)": {
			cfg: mustObjectValue(t, settingsAttrTypes, map[string]attr.Value{
				"quota": mustObjectValue(t, quotaAttrTypes, map[string]attr.Value{
					"active_time_seconds":  types.Int64Null(),
					"compute_time_seconds": types.Int64Null(),
					"written_data_bytes":   types.Int64Null(),
					"data_transfer_bytes":  types.Int64Null(),
					"logical_size_bytes":   types.Int64Null(),
				}),
				"allowed_ips":                types.ObjectNull(allowedIpsAttrTypes),
				"enable_logical_replication": types.BoolNull(),
				"maintenance_window":         types.ObjectNull(maintenanceWindowAttrTypes),
				"block_public_connections":   types.BoolNull(),
				"block_vpc_connections":      types.BoolNull(),
				"audit_log_level":            types.StringNull(),
				"hipaa":                      types.BoolNull(),
				"preload_libraries":          types.ObjectNull(preloadLibrariesAttrTypes),
			}),
		},
		"allowed_ips configured but empty (known, all leaves null)": {
			cfg: mustObjectValue(t, settingsAttrTypes, map[string]attr.Value{
				"quota": types.ObjectNull(quotaAttrTypes),
				"allowed_ips": mustObjectValue(t, allowedIpsAttrTypes, map[string]attr.Value{
					"ips":                     types.ListNull(types.StringType),
					"protected_branches_only": types.BoolNull(),
				}),
				"enable_logical_replication": types.BoolNull(),
				"maintenance_window":         types.ObjectNull(maintenanceWindowAttrTypes),
				"block_public_connections":   types.BoolNull(),
				"block_vpc_connections":      types.BoolNull(),
				"audit_log_level":            types.StringNull(),
				"hipaa":                      types.BoolNull(),
				"preload_libraries":          types.ObjectNull(preloadLibrariesAttrTypes),
			}),
		},
		"maintenance_window configured but empty (known, all leaves null)": {
			cfg: mustObjectValue(t, settingsAttrTypes, map[string]attr.Value{
				"quota":                      types.ObjectNull(quotaAttrTypes),
				"allowed_ips":                types.ObjectNull(allowedIpsAttrTypes),
				"enable_logical_replication": types.BoolNull(),
				"maintenance_window": mustObjectValue(t, maintenanceWindowAttrTypes, map[string]attr.Value{
					"weekdays":   types.SetNull(types.Int64Type),
					"start_time": types.StringNull(),
					"end_time":   types.StringNull(),
				}),
				"block_public_connections": types.BoolNull(),
				"block_vpc_connections":    types.BoolNull(),
				"audit_log_level":          types.StringNull(),
				"hipaa":                    types.BoolNull(),
				"preload_libraries":        types.ObjectNull(preloadLibrariesAttrTypes),
			}),
		},
		"preload_libraries configured but empty (known, all leaves null)": {
			cfg: mustObjectValue(t, settingsAttrTypes, map[string]attr.Value{
				"quota":                      types.ObjectNull(quotaAttrTypes),
				"allowed_ips":                types.ObjectNull(allowedIpsAttrTypes),
				"enable_logical_replication": types.BoolNull(),
				"maintenance_window":         types.ObjectNull(maintenanceWindowAttrTypes),
				"block_public_connections":   types.BoolNull(),
				"block_vpc_connections":      types.BoolNull(),
				"audit_log_level":            types.StringNull(),
				"hipaa":                      types.BoolNull(),
				"preload_libraries": mustObjectValue(t, preloadLibrariesAttrTypes, map[string]attr.Value{
					"use_defaults":      types.BoolNull(),
					"enabled_libraries": types.ListNull(types.StringType),
				}),
			}),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var diags diag.Diagnostics
			settings, included := buildProjectSettingsRequest(ctx, plan, tt.cfg, &diags)
			if diags.HasError() {
				t.Errorf("unexpected diagnostics: %v", diags)
			}
			if included {
				t.Errorf("expected included=false, got true (settings=%+v)", settings)
			}
			if settings.Quota.Set {
				t.Errorf("expected quota not to be attached, got %+v", settings.Quota)
			}
			if settings.AllowedIps.Set {
				t.Errorf("expected allowed_ips not to be attached, got %+v", settings.AllowedIps)
			}
			if settings.PreloadLibraries.Set {
				t.Errorf("expected preload_libraries not to be attached, got %+v", settings.PreloadLibraries)
			}
			if settings.MaintenanceWindow.Set {
				t.Errorf("expected maintenance_window not to be attached, got %+v", settings.MaintenanceWindow)
			}
		})
	}
}

// TestBuildProjectSettingsRequest_UnknownConfigIncludedFromPlan is the
// regression test for finding 1 at the settings level (cascading through every
// nested object): an unknown settings config value must be treated as
// configured, with every leaf - including inside quota / allowed_ips /
// maintenance_window / preload_libraries - taken from plan.
func TestBuildProjectSettingsRequest_UnknownConfigIncludedFromPlan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := fullyPopulatedSettingsPlan(t)
	cfg := types.ObjectUnknown(settingsAttrTypes)

	var diags diag.Diagnostics
	settings, included := buildProjectSettingsRequest(ctx, plan, cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !included {
		t.Fatal("expected included=true when config is unknown")
	}

	if v, ok := settings.EnableLogicalReplication.Get(); !ok || !v {
		t.Errorf("expected enable_logical_replication=true from plan, got %+v", settings.EnableLogicalReplication)
	}
	if !settings.Quota.Set {
		t.Error("expected quota to be attached from plan")
	} else if v, ok := settings.Quota.Value.ActiveTimeSeconds.Get(); !ok || v != 1 {
		t.Errorf("expected quota.active_time_seconds=1 from plan, got %+v", settings.Quota.Value.ActiveTimeSeconds)
	}
	if !settings.AllowedIps.Set {
		t.Error("expected allowed_ips to be attached from plan")
	} else if len(settings.AllowedIps.Value.Ips) != 1 || settings.AllowedIps.Value.Ips[0] != "0.0.0.0/0" {
		t.Errorf("expected allowed_ips.ips=[0.0.0.0/0] from plan, got %+v", settings.AllowedIps.Value.Ips)
	}
	if !settings.MaintenanceWindow.Set {
		t.Error("expected maintenance_window to be attached from plan")
	} else if settings.MaintenanceWindow.Value.StartTime != "01:00" || len(settings.MaintenanceWindow.Value.Weekdays) != 1 {
		t.Errorf("expected maintenance_window from plan, got %+v", settings.MaintenanceWindow.Value)
	}
	if !settings.PreloadLibraries.Set {
		t.Error("expected preload_libraries to be attached from plan")
	} else if len(settings.PreloadLibraries.Value.EnabledLibraries) != 1 {
		t.Errorf("expected preload_libraries from plan, got %+v", settings.PreloadLibraries.Value)
	}
}

// TestBuildProjectSettingsRequest_ConfiguredLeafIncluded is a sanity check that a
// single configured leaf is included while everything else is omitted, matching
// the leaf-level gating exercised end-to-end in
// TestProjectResource_UpdateSendsConfiguredSettings.
func TestBuildProjectSettingsRequest_ConfiguredLeafIncluded(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := fullyPopulatedSettingsPlan(t)

	cfg := mustObjectValue(t, settingsAttrTypes, map[string]attr.Value{
		"quota":                      types.ObjectNull(quotaAttrTypes),
		"allowed_ips":                types.ObjectNull(allowedIpsAttrTypes),
		"enable_logical_replication": types.BoolValue(true),
		"maintenance_window":         types.ObjectNull(maintenanceWindowAttrTypes),
		"block_public_connections":   types.BoolNull(),
		"block_vpc_connections":      types.BoolNull(),
		"audit_log_level":            types.StringNull(),
		"hipaa":                      types.BoolNull(),
		"preload_libraries":          types.ObjectNull(preloadLibrariesAttrTypes),
	})

	var diags diag.Diagnostics
	settings, included := buildProjectSettingsRequest(ctx, plan, cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !included {
		t.Fatal("expected included=true")
	}
	v, ok := settings.EnableLogicalReplication.Get()
	if !ok || !v {
		t.Errorf("expected enable_logical_replication=true to be included, got %+v", settings.EnableLogicalReplication)
	}
	if settings.Quota.Set || settings.AllowedIps.Set || settings.MaintenanceWindow.Set || settings.PreloadLibraries.Set {
		t.Errorf("expected only enable_logical_replication to be included, got %+v", settings)
	}
}

// TestBuildProjectMaintenanceWindowRequest_PartialLeafConfiguredIncludesOnlyThat
// verifies leaf-level gating within maintenance_window specifically (finding 2):
// even when maintenance_window itself is "configured" (cfg object non-null), an
// individual child whose cfg leaf is null must not be included. In practice the
// AlsoRequires validators prevent a real partial config from reaching here, but
// the builder must not rely on that; it re-derives inclusion per leaf.
func TestBuildProjectMaintenanceWindowRequest_PartialLeafConfiguredIncludesOnlyThat(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	pm := &projectSettingsModel{
		MaintenanceWindow: mustObjectValue(t, maintenanceWindowAttrTypes, map[string]attr.Value{
			"weekdays":   mustSetValue(t, types.Int64Type, []attr.Value{types.Int64Value(3)}),
			"start_time": types.StringValue("01:00"),
			"end_time":   types.StringValue("02:00"),
		}),
	}
	cm := &projectSettingsModel{
		MaintenanceWindow: mustObjectValue(t, maintenanceWindowAttrTypes, map[string]attr.Value{
			"weekdays":   types.SetNull(types.Int64Type),
			"start_time": types.StringValue("01:00"),
			"end_time":   types.StringNull(),
		}),
	}

	var diags diag.Diagnostics
	settings := &neon.ProjectSettingsData{}
	included := buildProjectMaintenanceWindowRequest(ctx, false, pm, cm, settings, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !included {
		t.Fatal("expected included=true (start_time was configured)")
	}
	if !settings.MaintenanceWindow.Set {
		t.Fatal("expected maintenance_window to be attached")
	}
	mw := settings.MaintenanceWindow.Value
	if mw.StartTime != "01:00" {
		t.Errorf("expected start_time=01:00 from plan, got %q", mw.StartTime)
	}
	if mw.EndTime != "" {
		t.Errorf("expected end_time to be omitted (zero value), got %q", mw.EndTime)
	}
	if mw.Weekdays != nil {
		t.Errorf("expected weekdays to be omitted (nil), got %+v", mw.Weekdays)
	}
}
