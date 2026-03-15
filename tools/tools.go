//go:build tools

package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
	_ "github.com/ogen-go/ogen/cmd/ogen"
)

//go:generate make -C .. generate/api-client
//go:generate make -C .. generate/docs
