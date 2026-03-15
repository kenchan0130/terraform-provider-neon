## Motivation

<!-- Why is this change needed? What problem does it solve? Link related issues if any. -->
<!-- e.g., "Closes #123" or "Users reported that ..." -->

## What changed

<!-- Describe the changes concisely. Highlight key design decisions if any. -->

-

## Type of change

<!-- Check the one that applies. -->

- [ ] New resource / data source / action
- [ ] Enhancement to existing resource / data source / action
- [ ] Bug fix
- [ ] Documentation
- [ ] CI / Infrastructure
- [ ] Dependency update
- [ ] Other

## How to verify

<!-- Steps a reviewer can follow to verify the change works as expected. -->
<!-- e.g., "Apply the example in examples/resources/neon_project/resource.tf" -->

1.

## Checklist

- [ ] `go build ./...` passes
- [ ] `go test ./... -v -count=1 -race -shuffle=on` passes
- [ ] `golangci-lint run ./...` passes
- [ ] Generated docs are up to date (`go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name neon`)
- [ ] New resource/data source/action is registered in `provider.go` (if applicable)
- [ ] Example `.tf` file is added under `examples/` (if applicable)

## Additional context

<!-- Anything else reviewers should know: breaking changes, migration steps, follow-up work, etc. -->
