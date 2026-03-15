package testutil

import (
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/provider"
)

func ProtoV6ProviderFactories(httpClient *http.Client) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"neon": providerserver.NewProtocol6WithError(provider.NewWithHTTPClient("test", httpClient)()),
	}
}

func TestConfig(extra string) string {
	return `
provider "neon" {
  api_key  = "test-api-key"
  base_url = "https://neon.example.com/api/v2"
}
` + extra
}

func CheckResourceAttr(name, key, value string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttr(name, key, value)
}

// JSONResponder creates an httpmock responder that returns JSON with proper Content-Type header.
func JSONResponder(status int, body string) httpmock.Responder {
	return func(_ *http.Request) (*http.Response, error) {
		resp := httpmock.NewStringResponse(status, body)
		resp.Header.Set("Content-Type", "application/json")
		return resp, nil
	}
}
