# CLAUDE.md

## Project Overview
Terraform provider for Neon serverless Postgres. Uses Terraform Plugin Framework (not legacy SDK).
API client is auto-generated from OpenAPI spec using ogen (`internal/neon/` - DO NOT edit manually).

## Commands
- `go build ./...` - Build
- `go test ./... -v -count=1 -race -shuffle=on` - Run tests
- `go generate ./...` - Regenerate API client from OpenAPI spec AND generate docs (runs both ogen and tfplugindocs)
- `go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name neon` - Generate docs only
- `golangci-lint run ./...` - Lint

## Project Structure
- `internal/neon/` - Auto-generated ogen API client (DO NOT EDIT)
- `internal/provider/provider.go` - Provider registration (resources, data sources, actions, ephemeral resources)
- `internal/services/<domain>/<entity>/` - Resource/data source implementations
  - Domains: `project/`, `branch/`, `endpoint/`, `organization/`, `api_key/`, `region/`
- `internal/testutil/` - Test helpers (ProtoV6ProviderFactories, JSONResponder, CheckResourceAttr)
- `internal/planmodifiers/` - Shared custom plan modifiers (e.g. `UnknownOnResourceChange` for volatile computed attributes)
- `internal/neonerror/` - API error helpers (`IsNotFound` etc.)
- `examples/resources/neon_*/resource.tf` - Resource examples for doc generation
- `examples/data-sources/neon_*/data-source.tf` - Data source examples for doc generation
- `examples/actions/neon_*/action.tf` - Action examples for doc generation
- `tools/tools.go` - Tool dependencies and go:generate directives

## Implementation Patterns

### Resource Pattern
- Implement `resource.Resource`, `resource.ResourceWithConfigure`, `resource.ResourceWithImportState`
- Use `*neon.Client` from `req.ProviderData`
- Model struct with `tfsdk` tags, use `types.String`, `types.Int64`, `types.Bool`, etc.
- Handle Opt* fields: `if v, ok := field.Get(); ok { ... } else { types.StringNull() }`
- Import format: `{parent_id}/{child_id}` parsed with `strings.SplitN(req.ID, "/", N)`
- When a resource implements `ResourceWithImportState`, add `examples/resources/neon_<name>/import.sh` with the import command (e.g., `terraform import neon_<name>.example <id_format>`)
- Collection attributes: prefer `SetAttribute`/`SetNestedAttribute` over `ListAttribute`/`ListNestedAttribute` unless element ordering is semantically meaningful. Use List only when the API requires or returns ordered elements.

### CRUD Correctness Rules (violations of these caused real bugs — treat as a review checklist)
- **Read + 404**: when the resource (or its parent) is gone, call `resp.State.RemoveResource(ctx)` and return — never surface 404 as an error (it makes `terraform refresh` fail forever). Use `neonerror.IsNotFound(err)`. This applies to list-based Reads too (parent deleted → list endpoint 404s).
- **Delete + 404**: treat as success (resource already gone externally).
- **Create orphans**: once the Create API call succeeds, any subsequent failure (read-back, follow-up calls) must NOT return before saving at least the resource ID to state — otherwise the created resource is orphaned outside Terraform. Save partial state first, then report the error.
- **Update must send every updatable attribute**: if an Optional attribute can change and the update API supports it, send it in Update. If the update API does NOT support it, the attribute needs `RequiresReplace` — otherwise changing it plans an in-place update that can never converge ("Provider produced inconsistent result after apply"). Check the generated `*UpdateRequest*` structs in `internal/neon/` to see what the API actually accepts; don't assume `RequiresReplace` is needed when the API supports in-place update (forcing replacement of point-in-time resources like snapshots loses data).

### Plan Modifier Rules
- **Never put `UseStateForUnknown` on volatile Computed attributes** (`updated_at`, `current_state`, `pending_state`, `last_active`, etc. — anything the server changes on every update). Terraform core carries the prior state's known value into the plan; when Update writes the API's new value, apply fails with "Provider produced inconsistent result after apply". Use `planmodifiers.UnknownOnResourceChange()` from `internal/planmodifiers/` instead. `UseStateForUnknown` is only for immutable Computed attributes (`id`, `created_at`).
- **Optional+Computed cannot be cleared**: removing an Optional+Computed attribute from config produces no diff (prior state fills it), so "remove from config to unset" never works. If the API supports clearing a value (e.g. sending null), make the attribute Optional-only.
- **API-normalized inputs**: when the API canonicalizes a practitioner-supplied value (timestamps with different offsets/precision), keep the config's representation in state if it means the same value (see `timestampValuePreservingConfig` in `internal/services/branch/branch/resource.go`); writing the normalized form over a non-Computed attribute causes inconsistent-result errors.

### Data Conversion Rules
- **Timestamps**: always `t.Format(time.RFC3339)`. Never `time.Time.String()` — it produces a non-RFC3339 Go debug format that contradicts the schema docs.
- **Empty vs missing collections**: ogen decodes JSON `[]` as a non-nil empty slice and leaves the field nil when absent. Map `!= nil` → empty Set/List, nil → null. Collapsing `[]` to null breaks configs that specify empty collections.
- **Provider Configure**: check `IsUnknown()` on provider config attributes and error out explicitly; treating unknown as "set" silently breaks env-var fallbacks like `NEON_API_KEY`.

### Data Source Pattern
- Same as resource but implement `datasource.DataSource`, `datasource.DataSourceWithConfigure`
- All output fields are `Computed: true`
- For list data sources with query params, use `query` as `SingleNestedAttribute` (NOT a block)

### Action Pattern
- Implement `action.Action`, `action.ActionWithConfigure`
- Use `Invoke` method (not CRUD)

### Test Pattern
- Use `httpmock.NewMockTransport()` for HTTP mocking
- Base URL in tests: `https://neon.example.com/api/v2`
- Use `testutil.JSONResponder(statusCode, jsonString)` for mock responses
- Use `testutil.ProtoV6ProviderFactories(httpClient)` for provider setup
- Use `testutil.TestConfig(hcl)` for test configurations
- Check response decoder in `oas_response_decoders_gen.go` for correct HTTP status codes
- **Every resource test MUST include an Update step** (change an attribute, mock the PATCH). A codebase-wide audit found the worst bugs concentrated in resources whose tests only covered Create/Import — every in-place update of those resources failed at apply time, undetected.
- Also cover: Read-after-404 (refresh removes from state, use `ExpectNonEmptyPlan`), and import round-trip (`ImportStateVerify`)

## Dependency Management
- When introducing or updating any dependency (Go modules, GitHub Actions, tools, etc.), always look up the latest version using WebSearch/WebFetch before specifying it
- GitHub Actions are pinned by full commit SHA with version comment (e.g., `uses: actions/checkout@<sha> # v6.0.2`)

## Git Conventions
- All commits MUST include a `Signed-off-by` line for DCO (Developer Certificate of Origin) compliance. Always use `--signoff` (or `-s`) flag when committing (e.g., `git commit -s -m "message"`)

## Common Gotchas
- API response status codes vary: check `oas_response_decoders_gen.go` for each operation (e.g., Delete may expect 202, not 200)
- `ApiKeysListResponseItem.CreatedBy` is `ApiKeyCreatorData` (has `id`, `name`, `image` fields), NOT a simple UUID
- Resources without `id` field need `ImportStateVerifyIdentifierAttribute` in import tests
- Pagination types differ: `Pagination{Cursor string}` for projects vs `CursorPagination{Next OptString}` for branches/members
- `internal/neon/` files are regenerated by `go generate` - never edit them manually
- When adding a new resource/data source/action, must register in `provider.go` (imports + Resources()/DataSources()/Actions() lists)
- Docs are auto-generated by tfplugindocs from schema + example files; run `go generate ./...` after changes
