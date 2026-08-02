package project

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func mustObjectValue(t *testing.T, attrTypes map[string]attr.Type, attrs map[string]attr.Value) basetypes.ObjectValue {
	t.Helper()
	obj, diags := types.ObjectValue(attrTypes, attrs)
	if diags.HasError() {
		t.Fatalf("failed to build object value: %v", diags)
	}
	return obj
}

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
		"weekdays":   mustListValue(t, types.Int64Type, []attr.Value{types.Int64Value(1)}),
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

func mustListValue(t *testing.T, elemType attr.Type, elems []attr.Value) basetypes.ListValue {
	t.Helper()
	list, diags := types.ListValue(elemType, elems)
	if diags.HasError() {
		t.Fatalf("failed to build list value: %v", diags)
	}
	return list
}

// TestBuildDefaultEndpointSettingsRequest_UnconfiguredOrEmpty verifies that
// unknown, null, or empty-but-known config objects never produce diagnostics and
// never cause a request to be attached (no empty {} emitted).
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
		"unknown config": {
			cfg: types.ObjectUnknown(defaultEndpointSettingsAttrTypes),
		},
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

// TestBuildProjectSettingsRequest_UnconfiguredOrEmpty covers the same cases for
// the top-level settings object, and the EMPTY OBJECT BAN for each nested object
// (quota / allowed_ips / preload_libraries): an unconfigured or empty nested
// object must never be attached, since e.g. an empty allowed_ips object can mean
// "allow all IPs" server-side.
func TestBuildProjectSettingsRequest_UnconfiguredOrEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	plan := fullyPopulatedSettingsPlan(t)

	tests := map[string]struct {
		cfg basetypes.ObjectValue
	}{
		"unknown config": {
			cfg: types.ObjectUnknown(settingsAttrTypes),
		},
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
