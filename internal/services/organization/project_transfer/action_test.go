package project_transfer_test

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
	"github.com/kenchan0130/terraform-provider-neon/internal/services/organization/project_transfer"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

func setupAction(t *testing.T, transport *httpmock.MockTransport) action.Action {
	t.Helper()

	httpClient := &http.Client{Transport: transport}
	secSource := &testSecuritySource{}
	client, err := neon.NewClient("https://neon.example.com/api/v2", secSource, neon.WithClient(httpClient))
	if err != nil {
		t.Fatalf("failed to create neon client: %v", err)
	}

	a := project_transfer.NewAction()

	configResp := &action.ConfigureResponse{}
	a.(action.ActionWithConfigure).Configure(context.Background(), action.ConfigureRequest{
		ProviderData: client,
	}, configResp)
	if configResp.Diagnostics.HasError() {
		t.Fatalf("configure failed: %s", configResp.Diagnostics.Errors())
	}

	return a
}

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

func newInvokeConfig(sourceOrgID, destOrgID string, projectIDs []string) tfsdk.Config {
	s := getActionSchema()
	schemaType := s.Type().TerraformType(context.Background())

	projectIDValues := make([]tftypes.Value, len(projectIDs))
	for i, id := range projectIDs {
		projectIDValues[i] = tftypes.NewValue(tftypes.String, id)
	}

	return tfsdk.Config{
		Schema: s,
		Raw: tftypes.NewValue(schemaType, map[string]tftypes.Value{
			"source_org_id":      tftypes.NewValue(tftypes.String, sourceOrgID),
			"destination_org_id": tftypes.NewValue(tftypes.String, destOrgID),
			"project_ids":        tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, projectIDValues),
		}),
	}
}

func getActionSchema() actionschema.Schema {
	a := project_transfer.NewAction()
	schemaResp := &action.SchemaResponse{}
	a.Schema(context.Background(), action.SchemaRequest{}, schemaResp)
	return schemaResp.Schema
}

func TestProjectTransferAction_Invoke(t *testing.T) {
	transport := httpmock.NewMockTransport()

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/organizations/org-source-001/projects/transfer",
		testutil.JSONResponder(200, `{}`),
	)

	a := setupAction(t, transport)

	resp := &action.InvokeResponse{}
	a.Invoke(context.Background(), action.InvokeRequest{
		Config: newInvokeConfig("org-source-001", "org-dest-001", []string{"proj-001", "proj-002"}),
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}
}

func TestProjectTransferAction_LimitsError(t *testing.T) {
	transport := httpmock.NewMockTransport()

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/organizations/org-source-001/projects/transfer",
		testutil.JSONResponder(422, `{"limits":[{"name":"projects_count","expected":"10","actual":"15"}]}`),
	)

	a := setupAction(t, transport)

	resp := &action.InvokeResponse{}
	a.Invoke(context.Background(), action.InvokeRequest{
		Config: newInvokeConfig("org-source-001", "org-dest-001", []string{"proj-001"}),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error but got none")
	}
}

func TestProjectTransferAction_IntegrationError(t *testing.T) {
	transport := httpmock.NewMockTransport()

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/organizations/org-source-001/projects/transfer",
		testutil.JSONResponder(409, `{"projects":[{"id":"proj-001","integration":"vercel"}]}`),
	)

	a := setupAction(t, transport)

	resp := &action.InvokeResponse{}
	a.Invoke(context.Background(), action.InvokeRequest{
		Config: newInvokeConfig("org-source-001", "org-dest-001", []string{"proj-001"}),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error but got none")
	}
}

func TestProjectTransferAction_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/organizations/org-source-001/projects/transfer",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	a := setupAction(t, transport)

	resp := &action.InvokeResponse{}
	a.Invoke(context.Background(), action.InvokeRequest{
		Config: newInvokeConfig("org-source-001", "org-dest-001", []string{"proj-001"}),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error but got none")
	}
}
