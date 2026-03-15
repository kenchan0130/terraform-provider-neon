package provider_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/provider"
)

func TestProvider_MissingAPIKey(t *testing.T) {
	t.Setenv("NEON_API_KEY", "")

	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"neon": providerserver.NewProtocol6WithError(provider.NewWithHTTPClient("test", httpClient)()),
		},
		Steps: []resource.TestStep{
			{
				Config: `
provider "neon" {}

data "neon_project" "test" {
  id = "test-project-id"
}
`,
				ExpectError: regexp.MustCompile(`Missing API Key`),
			},
		},
	})
}
