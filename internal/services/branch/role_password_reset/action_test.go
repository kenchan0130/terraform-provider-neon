package role_password_reset_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/role_password_reset"
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

func setupAction(t *testing.T, transport *httpmock.MockTransport) action.Action {
	t.Helper()

	httpClient := &http.Client{Transport: transport}
	client, err := neon.NewClient("https://neon.example.com/api/v2", &testSecuritySource{}, neon.WithClient(httpClient))
	if err != nil {
		t.Fatalf("failed to create neon client: %v", err)
	}

	a := role_password_reset.NewAction()

	configResp := &action.ConfigureResponse{}
	a.(action.ActionWithConfigure).Configure(context.Background(), action.ConfigureRequest{
		ProviderData: client,
	}, configResp)
	if configResp.Diagnostics.HasError() {
		t.Fatalf("configure failed: %s", configResp.Diagnostics.Errors())
	}

	return a
}

func getActionSchema() actionschema.Schema {
	a := role_password_reset.NewAction()
	schemaResp := &action.SchemaResponse{}
	a.Schema(context.Background(), action.SchemaRequest{}, schemaResp)
	return schemaResp.Schema
}

func newInvokeConfig(projectID, branchID, roleName string) tfsdk.Config {
	s := getActionSchema()
	schemaType := s.Type().TerraformType(context.Background())

	return tfsdk.Config{
		Schema: s,
		Raw: tftypes.NewValue(schemaType, map[string]tftypes.Value{
			"project_id": tftypes.NewValue(tftypes.String, projectID),
			"branch_id":  tftypes.NewValue(tftypes.String, branchID),
			"role_name":  tftypes.NewValue(tftypes.String, roleName),
		}),
	}
}

func TestRolePasswordResetAction_Invoke(t *testing.T) {
	transport := httpmock.NewMockTransport()

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/proj-001/branches/br-001/roles/myrole/reset_password",
		testutil.JSONResponder(200, `{
			"role": {"branch_id":"br-001","name":"myrole","protected":false,"created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z"},
			"operations": []
		}`),
	)

	a := setupAction(t, transport)

	resp := &action.InvokeResponse{}
	a.Invoke(context.Background(), action.InvokeRequest{
		Config: newInvokeConfig("proj-001", "br-001", "myrole"),
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}
}

func TestRolePasswordResetAction_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/proj-001/branches/br-001/roles/myrole/reset_password",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	a := setupAction(t, transport)

	resp := &action.InvokeResponse{}
	a.Invoke(context.Background(), action.InvokeRequest{
		Config: newInvokeConfig("proj-001", "br-001", "myrole"),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error but got none")
	}
}
