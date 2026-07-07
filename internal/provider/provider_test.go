package provider_test

import (
	"context"
	"net/http"
	"regexp"
	"testing"

	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

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

func TestProvider_UnknownAPIKey(t *testing.T) {
	t.Setenv("NEON_API_KEY", "test-api-key")

	p := provider.New("test")()

	schemaResp := &fwprovider.SchemaResponse{}
	p.Schema(context.Background(), fwprovider.SchemaRequest{}, schemaResp)

	schemaType := schemaResp.Schema.Type().TerraformType(context.Background())

	configureResp := &fwprovider.ConfigureResponse{}
	p.Configure(context.Background(), fwprovider.ConfigureRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw: tftypes.NewValue(schemaType, map[string]tftypes.Value{
				"api_key":  tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				"base_url": tftypes.NewValue(tftypes.String, nil),
			}),
		},
	}, configureResp)

	if !configureResp.Diagnostics.HasError() {
		t.Fatal("expected error for unknown api_key, but got none")
	}

	found := false
	for _, d := range configureResp.Diagnostics.Errors() {
		if regexp.MustCompile(`Unknown Neon API Key`).MatchString(d.Summary()) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'Unknown Neon API Key' diagnostic, got: %v", configureResp.Diagnostics.Errors())
	}
}
