// Package planmodifiers contains custom plan modifiers shared across
// terraform-provider-neon resources.
package planmodifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const unknownOnResourceChangeDescription = "Marks the attribute as unknown when the resource is being updated, since the API changes this value on every update."

// UnknownOnResourceChange returns a plan modifier that marks a volatile
// Computed string attribute (e.g. updated_at) as unknown whenever the
// resource has an actual change, while leaving it alone on create/destroy
// or when there is no change at all.
//
// This is intended for Computed-only attributes that the API mutates on
// every update (timestamps, etc.). Terraform core normally carries the
// prior state's known value into the plan for such attributes; if the
// Update implementation then writes a different value into state,
// Terraform reports "Provider produced inconsistent result after apply".
// Marking the value unknown during an update avoids that inconsistency.
func UnknownOnResourceChange() planmodifier.String {
	return unknownOnResourceChangeModifier{}
}

type unknownOnResourceChangeModifier struct{}

func (m unknownOnResourceChangeModifier) Description(_ context.Context) string {
	return unknownOnResourceChangeDescription
}

func (m unknownOnResourceChangeModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m unknownOnResourceChangeModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Resource is being created: there is no prior state to compare against.
	if req.State.Raw.IsNull() {
		return
	}

	// Resource is being destroyed: there is no plan to modify.
	if req.Plan.Raw.IsNull() {
		return
	}

	// No change to the resource at all: don't create a diff for a
	// Computed-only attribute that never appears in config.
	if req.Plan.Raw.Equal(req.State.Raw) {
		return
	}

	// The resource has some other real change; the volatile attribute will
	// be recomputed by the API, so mark it unknown.
	resp.PlanValue = types.StringUnknown()
}

// UnknownOnResourceChangeInt64 returns the Int64 equivalent of
// UnknownOnResourceChange. See that function's documentation for details.
func UnknownOnResourceChangeInt64() planmodifier.Int64 {
	return unknownOnResourceChangeInt64Modifier{}
}

type unknownOnResourceChangeInt64Modifier struct{}

func (m unknownOnResourceChangeInt64Modifier) Description(_ context.Context) string {
	return unknownOnResourceChangeDescription
}

func (m unknownOnResourceChangeInt64Modifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m unknownOnResourceChangeInt64Modifier) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	// Resource is being created: there is no prior state to compare against.
	if req.State.Raw.IsNull() {
		return
	}

	// Resource is being destroyed: there is no plan to modify.
	if req.Plan.Raw.IsNull() {
		return
	}

	// No change to the resource at all: don't create a diff for a
	// Computed-only attribute that never appears in config.
	if req.Plan.Raw.Equal(req.State.Raw) {
		return
	}

	// The resource has some other real change; the volatile attribute will
	// be recomputed by the API, so mark it unknown.
	resp.PlanValue = types.Int64Unknown()
}

// UnknownOnResourceChangeList returns the List equivalent of
// UnknownOnResourceChange. See that function's documentation for details.
func UnknownOnResourceChangeList() planmodifier.List {
	return unknownOnResourceChangeListModifier{}
}

type unknownOnResourceChangeListModifier struct{}

func (m unknownOnResourceChangeListModifier) Description(_ context.Context) string {
	return unknownOnResourceChangeDescription
}

func (m unknownOnResourceChangeListModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m unknownOnResourceChangeListModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	// Resource is being created: there is no prior state to compare against.
	if req.State.Raw.IsNull() {
		return
	}

	// Resource is being destroyed: there is no plan to modify.
	if req.Plan.Raw.IsNull() {
		return
	}

	// No change to the resource at all: don't create a diff for a
	// Computed-only attribute that never appears in config.
	if req.Plan.Raw.Equal(req.State.Raw) {
		return
	}

	// The resource has some other real change; the volatile attribute will
	// be recomputed by the API, so mark it unknown.
	resp.PlanValue = types.ListUnknown(resp.PlanValue.ElementType(ctx))
}
