# Terraform Provider Neon

A Terraform provider for [Neon](https://neon.tech/) serverless Postgres. Manage projects, branches, endpoints, databases, roles, organizations, and more through Terraform.

## Documentation

Full, comprehensive documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/kenchan0130/neon). [API documentation](https://api-docs.neon.tech/reference/getting-started-with-neon-api) is also available for non-Terraform or service specific information.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads)
- [Go](https://golang.org/doc/install)

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the `make` command:

```shell
make build
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

```terraform
terraform {
  required_providers {
    neon = {
      source  = "kenchan0130/neon"
      version = "~> 0.0.1"
    }
  }
}

provider "neon" {
  # api_key = "..." # Or set NEON_API_KEY environment variable
}
```

## Debugging the Provider

### Case 1. Local install

You can debug developing provider using following steps:

1. `make local_install`
1. Edit `~/.terraformrc` using the output comment
1. `cd examples/resources/neon_project`
1. `TF_LOG_PROVIDER=debug terraform apply`

### Case 2. Using Visual Studio Code

You can debug developing provider using following steps:

1. Launch your Visual Studio Code app
1. Select `Debug Terraform Provier` configuration and start a debugging session from "Run and Debug" view
1. Copy a `TF_REATTACH_PROVIDERS={...}` output from "Debug Console" tab
1. `cd examples/resources/neon_project`
1. `TF_REATTACH_PROVIDERS={...} TF_LOG_PROVIDER=debug terraform apply`

## Release

We use release management by [tagpr](https://github.com/Songmu/tagpr). When merging tagpr PR, next version would be released by github-actions.

## Contribution

See also [CONTRIBUTING.md](CONTRIBUTING.md).

### DCO Sign-Off Methods

The sign-off is a simple line at the end of the explanation for the patch, which certifies that you wrote it or otherwise have the right to pass it on as an open-source patch.

The DCO requires a sign-off message in the following format appear on each commit in the pull request:

```txt
Signed-off-by: Sample Developer sample@example.com
```

The text can either be manually added to your commit body, or you can add either `-s` or `--signoff` to your usual `git` commit commands.

#### Auto sign-off

The following method is examples only and are not mandatory.

```sh
touch .git/hooks/prepare-commit-msg
chmod +x .git/hooks/prepare-commit-msg
```

Edit the `prepare-commit-msg` file like:

```sh
#!/bin/sh

name=$(git config user.name)
email=$(git config user.email)

if [ -z "${name}" ]; then
  echo "empty git config user.name"
  exit 1
fi

if [ -z "${email}" ]; then
  echo "empty git config user.email"
  exit 1
fi

git interpret-trailers --if-exists doNothing --trailer \
    "Signed-off-by: ${name} <${email}>" \
    --in-place "$1"
```
