package connection_uri_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	ephemeralschema "github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/connection_uri"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

type testSecuritySource struct{}

func (s *testSecuritySource) BearerAuth(_ context.Context, _ string) (neon.BearerAuth, error) {
	return neon.BearerAuth{Token: "test-api-key"}, nil
}

func (s *testSecuritySource) CookieAuth(_ context.Context, _ string) (neon.CookieAuth, error) {
	return neon.CookieAuth{}, nil
}

func (s *testSecuritySource) TokenCookieAuth(_ context.Context, _ string) (neon.TokenCookieAuth, error) {
	return neon.TokenCookieAuth{}, nil
}

func setupEphemeral(t *testing.T, transport *httpmock.MockTransport) ephemeral.EphemeralResource {
	t.Helper()

	httpClient := &http.Client{Transport: transport}
	client, err := neon.NewClient("https://neon.example.com/api/v2", &testSecuritySource{}, neon.WithClient(httpClient))
	if err != nil {
		t.Fatalf("failed to create neon client: %v", err)
	}

	e := connection_uri.NewEphemeralResource()

	configResp := &ephemeral.ConfigureResponse{}
	e.(ephemeral.EphemeralResourceWithConfigure).Configure(context.Background(), ephemeral.ConfigureRequest{
		ProviderData: client,
	}, configResp)
	if configResp.Diagnostics.HasError() {
		t.Fatalf("configure failed: %s", configResp.Diagnostics.Errors())
	}

	return e
}

func getEphemeralSchema() ephemeralschema.Schema {
	e := connection_uri.NewEphemeralResource()
	schemaResp := &ephemeral.SchemaResponse{}
	e.Schema(context.Background(), ephemeral.SchemaRequest{}, schemaResp)
	return schemaResp.Schema
}

func newOpenConfig(projectID, databaseName, roleName string) tfsdk.Config {
	s := getEphemeralSchema()
	schemaType := s.Type().TerraformType(context.Background())

	return tfsdk.Config{
		Schema: s,
		Raw: tftypes.NewValue(schemaType, map[string]tftypes.Value{
			"project_id":    tftypes.NewValue(tftypes.String, projectID),
			"branch_id":     tftypes.NewValue(tftypes.String, nil),
			"endpoint_id":   tftypes.NewValue(tftypes.String, nil),
			"database_name": tftypes.NewValue(tftypes.String, databaseName),
			"role_name":     tftypes.NewValue(tftypes.String, roleName),
			"pooled":        tftypes.NewValue(tftypes.Bool, nil),
			"uri":           tftypes.NewValue(tftypes.String, nil),
		}),
	}
}

func TestConnectionURIEphemeral_Open(t *testing.T) {
	transport := httpmock.NewMockTransport()

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/connection_uri",
		testutil.JSONResponder(200, `{"uri":"postgres://user:pass@host/dbname"}`),
	)

	e := setupEphemeral(t, transport)

	resp := &ephemeral.OpenResponse{}
	resp.Result = tfsdk.EphemeralResultData{
		Schema: getEphemeralSchema(),
	}
	e.Open(context.Background(), ephemeral.OpenRequest{
		Config: newOpenConfig("test-project-id", "dbname", "user"),
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}
}

func TestConnectionURIEphemeral_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/connection_uri",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	e := setupEphemeral(t, transport)

	resp := &ephemeral.OpenResponse{}
	resp.Result = tfsdk.EphemeralResultData{
		Schema: getEphemeralSchema(),
	}
	e.Open(context.Background(), ephemeral.OpenRequest{
		Config: newOpenConfig("test-project-id", "dbname", "user"),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error but got none")
	}
}
