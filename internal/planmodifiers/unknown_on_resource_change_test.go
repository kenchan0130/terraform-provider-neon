package planmodifiers

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var testSchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Required: true,
		},
		"updated_at": schema.StringAttribute{
			Computed: true,
		},
	},
}

var testSchemaType = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"id":         tftypes.String,
		"updated_at": tftypes.String,
	},
}

func newState(t *testing.T, id, updatedAt string, updatedAtKnown bool) tfsdk.State {
	t.Helper()

	updatedAtValue := tftypes.NewValue(tftypes.String, updatedAt)
	if !updatedAtKnown {
		updatedAtValue = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	}

	raw := tftypes.NewValue(testSchemaType, map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, id),
		"updated_at": updatedAtValue,
	})

	return tfsdk.State{
		Raw:    raw,
		Schema: testSchema,
	}
}

func newNullState() tfsdk.State {
	return tfsdk.State{
		Raw:    tftypes.NewValue(testSchemaType, nil),
		Schema: testSchema,
	}
}

func newPlan(t *testing.T, id, updatedAt string, updatedAtKnown bool) tfsdk.Plan {
	t.Helper()

	updatedAtValue := tftypes.NewValue(tftypes.String, updatedAt)
	if !updatedAtKnown {
		updatedAtValue = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	}

	raw := tftypes.NewValue(testSchemaType, map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, id),
		"updated_at": updatedAtValue,
	})

	return tfsdk.Plan{
		Raw:    raw,
		Schema: testSchema,
	}
}

func newNullPlan() tfsdk.Plan {
	return tfsdk.Plan{
		Raw:    tftypes.NewValue(testSchemaType, nil),
		Schema: testSchema,
	}
}

func TestUnknownOnResourceChangePlanModifyString(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		state         tfsdk.State
		plan          tfsdk.Plan
		stateValue    types.String
		planValue     types.String
		expectedValue types.String
	}{
		"create: state is null": {
			state:         newNullState(),
			plan:          newPlan(t, "id-1", "", false),
			stateValue:    types.StringNull(),
			planValue:     types.StringUnknown(),
			expectedValue: types.StringUnknown(),
		},
		"destroy: plan is null": {
			state:         newState(t, "id-1", "2024-01-01T00:00:00Z", true),
			plan:          newNullPlan(),
			stateValue:    types.StringValue("2024-01-01T00:00:00Z"),
			planValue:     types.StringNull(),
			expectedValue: types.StringNull(),
		},
		"no change: plan equals state": {
			state:         newState(t, "id-1", "2024-01-01T00:00:00Z", true),
			plan:          newPlan(t, "id-1", "2024-01-01T00:00:00Z", true),
			stateValue:    types.StringValue("2024-01-01T00:00:00Z"),
			planValue:     types.StringValue("2024-01-01T00:00:00Z"),
			expectedValue: types.StringValue("2024-01-01T00:00:00Z"),
		},
		"update: other attribute changed": {
			state:         newState(t, "id-1", "2024-01-01T00:00:00Z", true),
			plan:          newPlan(t, "id-2", "2024-01-01T00:00:00Z", true),
			stateValue:    types.StringValue("2024-01-01T00:00:00Z"),
			planValue:     types.StringValue("2024-01-01T00:00:00Z"),
			expectedValue: types.StringUnknown(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.StringRequest{
				State:      tc.state,
				Plan:       tc.plan,
				StateValue: tc.stateValue,
				PlanValue:  tc.planValue,
			}
			resp := &planmodifier.StringResponse{
				PlanValue: tc.planValue,
			}

			UnknownOnResourceChange().PlanModifyString(context.Background(), req, resp)

			if !resp.PlanValue.Equal(tc.expectedValue) {
				t.Errorf("expected plan value %v, got %v", tc.expectedValue, resp.PlanValue)
			}
		})
	}
}

var testInt64SchemaType = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"id":    tftypes.String,
		"count": tftypes.Number,
	},
}

var testInt64Schema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Required: true,
		},
		"count": schema.Int64Attribute{
			Computed: true,
		},
	},
}

func newInt64State(t *testing.T, id string, count int64, countKnown bool) tfsdk.State {
	t.Helper()

	countValue := tftypes.NewValue(tftypes.Number, count)
	if !countKnown {
		countValue = tftypes.NewValue(tftypes.Number, tftypes.UnknownValue)
	}

	raw := tftypes.NewValue(testInt64SchemaType, map[string]tftypes.Value{
		"id":    tftypes.NewValue(tftypes.String, id),
		"count": countValue,
	})

	return tfsdk.State{
		Raw:    raw,
		Schema: testInt64Schema,
	}
}

func newNullInt64State() tfsdk.State {
	return tfsdk.State{
		Raw:    tftypes.NewValue(testInt64SchemaType, nil),
		Schema: testInt64Schema,
	}
}

func newInt64Plan(t *testing.T, id string, count int64, countKnown bool) tfsdk.Plan {
	t.Helper()

	countValue := tftypes.NewValue(tftypes.Number, count)
	if !countKnown {
		countValue = tftypes.NewValue(tftypes.Number, tftypes.UnknownValue)
	}

	raw := tftypes.NewValue(testInt64SchemaType, map[string]tftypes.Value{
		"id":    tftypes.NewValue(tftypes.String, id),
		"count": countValue,
	})

	return tfsdk.Plan{
		Raw:    raw,
		Schema: testInt64Schema,
	}
}

func newNullInt64Plan() tfsdk.Plan {
	return tfsdk.Plan{
		Raw:    tftypes.NewValue(testInt64SchemaType, nil),
		Schema: testInt64Schema,
	}
}

func TestUnknownOnResourceChangePlanModifyInt64(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		state         tfsdk.State
		plan          tfsdk.Plan
		stateValue    types.Int64
		planValue     types.Int64
		expectedValue types.Int64
	}{
		"create: state is null": {
			state:         newNullInt64State(),
			plan:          newInt64Plan(t, "id-1", 0, false),
			stateValue:    types.Int64Null(),
			planValue:     types.Int64Unknown(),
			expectedValue: types.Int64Unknown(),
		},
		"destroy: plan is null": {
			state:         newInt64State(t, "id-1", 1, true),
			plan:          newNullInt64Plan(),
			stateValue:    types.Int64Value(1),
			planValue:     types.Int64Null(),
			expectedValue: types.Int64Null(),
		},
		"no change: plan equals state": {
			state:         newInt64State(t, "id-1", 1, true),
			plan:          newInt64Plan(t, "id-1", 1, true),
			stateValue:    types.Int64Value(1),
			planValue:     types.Int64Value(1),
			expectedValue: types.Int64Value(1),
		},
		"update: other attribute changed": {
			state:         newInt64State(t, "id-1", 1, true),
			plan:          newInt64Plan(t, "id-2", 1, true),
			stateValue:    types.Int64Value(1),
			planValue:     types.Int64Value(1),
			expectedValue: types.Int64Unknown(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.Int64Request{
				State:      tc.state,
				Plan:       tc.plan,
				StateValue: tc.stateValue,
				PlanValue:  tc.planValue,
			}
			resp := &planmodifier.Int64Response{
				PlanValue: tc.planValue,
			}

			UnknownOnResourceChangeInt64().PlanModifyInt64(context.Background(), req, resp)

			if !resp.PlanValue.Equal(tc.expectedValue) {
				t.Errorf("expected plan value %v, got %v", tc.expectedValue, resp.PlanValue)
			}
		})
	}
}

func TestUnknownOnResourceChangeInt64Description(t *testing.T) {
	t.Parallel()

	m := UnknownOnResourceChangeInt64()
	if m.Description(context.Background()) == "" {
		t.Error("expected non-empty description")
	}
	if m.MarkdownDescription(context.Background()) == "" {
		t.Error("expected non-empty markdown description")
	}
}

var testListSchemaType = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"id":    tftypes.String,
		"items": tftypes.List{ElementType: tftypes.String},
	},
}

var testListSchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Required: true,
		},
		"items": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
		},
	},
}

func newListState(t *testing.T, id string, items []string, itemsKnown bool) tfsdk.State {
	t.Helper()

	itemsValue := listValue(items)
	if !itemsKnown {
		itemsValue = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, tftypes.UnknownValue)
	}

	raw := tftypes.NewValue(testListSchemaType, map[string]tftypes.Value{
		"id":    tftypes.NewValue(tftypes.String, id),
		"items": itemsValue,
	})

	return tfsdk.State{
		Raw:    raw,
		Schema: testListSchema,
	}
}

func newNullListState() tfsdk.State {
	return tfsdk.State{
		Raw:    tftypes.NewValue(testListSchemaType, nil),
		Schema: testListSchema,
	}
}

func newListPlan(t *testing.T, id string, items []string, itemsKnown bool) tfsdk.Plan {
	t.Helper()

	itemsValue := listValue(items)
	if !itemsKnown {
		itemsValue = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, tftypes.UnknownValue)
	}

	raw := tftypes.NewValue(testListSchemaType, map[string]tftypes.Value{
		"id":    tftypes.NewValue(tftypes.String, id),
		"items": itemsValue,
	})

	return tfsdk.Plan{
		Raw:    raw,
		Schema: testListSchema,
	}
}

func newNullListPlan() tfsdk.Plan {
	return tfsdk.Plan{
		Raw:    tftypes.NewValue(testListSchemaType, nil),
		Schema: testListSchema,
	}
}

func listValue(items []string) tftypes.Value {
	elems := make([]tftypes.Value, len(items))
	for i, item := range items {
		elems[i] = tftypes.NewValue(tftypes.String, item)
	}
	return tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, elems)
}

func mustListTypeValue(t *testing.T, items []string) types.List {
	t.Helper()

	elems := make([]attr.Value, len(items))
	for i, item := range items {
		elems[i] = types.StringValue(item)
	}
	list, diags := types.ListValue(types.StringType, elems)
	if diags.HasError() {
		t.Fatalf("unexpected error building list value: %v", diags)
	}
	return list
}

func TestUnknownOnResourceChangePlanModifyList(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		state         tfsdk.State
		plan          tfsdk.Plan
		stateValue    types.List
		planValue     types.List
		expectedValue types.List
	}{
		"create: state is null": {
			state:         newNullListState(),
			plan:          newListPlan(t, "id-1", nil, false),
			stateValue:    types.ListNull(types.StringType),
			planValue:     types.ListUnknown(types.StringType),
			expectedValue: types.ListUnknown(types.StringType),
		},
		"destroy: plan is null": {
			state:         newListState(t, "id-1", []string{"a"}, true),
			plan:          newNullListPlan(),
			stateValue:    mustListTypeValue(t, []string{"a"}),
			planValue:     types.ListNull(types.StringType),
			expectedValue: types.ListNull(types.StringType),
		},
		"no change: plan equals state": {
			state:         newListState(t, "id-1", []string{"a"}, true),
			plan:          newListPlan(t, "id-1", []string{"a"}, true),
			stateValue:    mustListTypeValue(t, []string{"a"}),
			planValue:     mustListTypeValue(t, []string{"a"}),
			expectedValue: mustListTypeValue(t, []string{"a"}),
		},
		"update: other attribute changed": {
			state:         newListState(t, "id-1", []string{"a"}, true),
			plan:          newListPlan(t, "id-2", []string{"a"}, true),
			stateValue:    mustListTypeValue(t, []string{"a"}),
			planValue:     mustListTypeValue(t, []string{"a"}),
			expectedValue: types.ListUnknown(types.StringType),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.ListRequest{
				State:      tc.state,
				Plan:       tc.plan,
				StateValue: tc.stateValue,
				PlanValue:  tc.planValue,
			}
			resp := &planmodifier.ListResponse{
				PlanValue: tc.planValue,
			}

			UnknownOnResourceChangeList().PlanModifyList(context.Background(), req, resp)

			if !resp.PlanValue.Equal(tc.expectedValue) {
				t.Errorf("expected plan value %v, got %v", tc.expectedValue, resp.PlanValue)
			}
		})
	}
}

func TestUnknownOnResourceChangeListDescription(t *testing.T) {
	t.Parallel()

	m := UnknownOnResourceChangeList()
	if m.Description(context.Background()) == "" {
		t.Error("expected non-empty description")
	}
	if m.MarkdownDescription(context.Background()) == "" {
		t.Error("expected non-empty markdown description")
	}
}

func TestUnknownOnResourceChangeDescription(t *testing.T) {
	t.Parallel()

	m := UnknownOnResourceChange()
	if m.Description(context.Background()) == "" {
		t.Error("expected non-empty description")
	}
	if m.MarkdownDescription(context.Background()) == "" {
		t.Error("expected non-empty markdown description")
	}
}
