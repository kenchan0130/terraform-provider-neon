//go:build tools

package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
	_ "github.com/ogen-go/ogen/cmd/ogen"
)

//go:generate go run github.com/ogen-go/ogen/cmd/ogen --target ../internal/neon -package neon --clean https://neon.com/api_spec/release/v2.json
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name neon
