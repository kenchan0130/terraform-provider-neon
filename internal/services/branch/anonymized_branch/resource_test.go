package anonymized_branch_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const branchJSON = `{
	"id": "br-anon-001",
	"project_id": "test-project-id",
	"name": "anon-branch",
	"parent_id": "br-parent-001",
	"current_state": "ready",
	"state_changed_at": "2025-01-01T00:00:00Z",
	"creation_source": "console",
	"primary": false,
	"default": false,
	"protected": false,
	"cpu_used_sec": 0,
	"compute_time_seconds": 0,
	"active_time_seconds": 0,
	"written_data_bytes": 0,
	"data_transfer_bytes": 0,
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z"
}`

const maskingRulesJSON = `{
	"masking_rules": [
		{
			"database_name": "mydb",
			"schema_name": "public",
			"table_name": "users",
			"column_name": "email",
			"masking_function": "anon.fake_email()"
		}
	]
}`

const anonymizedStatusJSON = `{
	"project_id": "test-project-id",
	"branch_id": "br-anon-001",
	"state": "created",
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z"
}`

func setupAnonymizedBranchMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branch_anonymized",
		testutil.JSONResponder(201, `{
			"branch": `+branchJSON+`,
			"endpoints": [],
			"operations": [],
			"roles": [],
			"databases": [],
			"connection_uris": []
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-anon-001",
		testutil.JSONResponder(200, `{"branch": `+branchJSON+`, "annotation": {"object": {"type": "branch", "id": "br-anon-001"}, "value": {}}}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-anon-001/masking_rules",
		testutil.JSONResponder(200, maskingRulesJSON),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-anon-001/anonymized_status",
		testutil.JSONResponder(200, anonymizedStatusJSON),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id/branches/br-anon-001",
		testutil.JSONResponder(200, `{"branch": `+branchJSON+`, "operations": []}`),
	)
}

func TestAnonymizedBranchResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupAnonymizedBranchMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_anonymized_branch" "test" {
  project_id = "test-project-id"
  name       = "anon-branch"

  masking_rules {
    database_name    = "mydb"
    schema_name      = "public"
    table_name       = "users"
    column_name      = "email"
    masking_function = "anon.fake_email()"
  }
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_anonymized_branch.test", "id", "br-anon-001"),
					testutil.CheckResourceAttr("neon_anonymized_branch.test", "project_id", "test-project-id"),
					testutil.CheckResourceAttr("neon_anonymized_branch.test", "name", "anon-branch"),
					testutil.CheckResourceAttr("neon_anonymized_branch.test", "parent_id", "br-parent-001"),
					testutil.CheckResourceAttr("neon_anonymized_branch.test", "state", "created"),
					testutil.CheckResourceAttr("neon_anonymized_branch.test", "masking_rules.#", "1"),
					testutil.CheckResourceAttr("neon_anonymized_branch.test", "masking_rules.0.database_name", "mydb"),
					testutil.CheckResourceAttr("neon_anonymized_branch.test", "masking_rules.0.masking_function", "anon.fake_email()"),
				),
			},
		},
	})
}

func TestAnonymizedBranchResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupAnonymizedBranchMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_anonymized_branch" "test" {
  project_id = "test-project-id"
  name       = "anon-branch"

  masking_rules {
    database_name    = "mydb"
    schema_name      = "public"
    table_name       = "users"
    column_name      = "email"
    masking_function = "anon.fake_email()"
  }
}
`),
			},
			{
				ResourceName:            "neon_anonymized_branch.test",
				ImportState:             true,
				ImportStateId:           "test-project-id/br-anon-001",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"start_anonymization"},
			},
		},
	})
}

func TestAnonymizedBranchResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects/test-project-id/branch_anonymized",
		testutil.JSONResponder(500, `{"message":"internal error"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_anonymized_branch" "test" {
  project_id = "test-project-id"
  name       = "anon-branch"

  masking_rules {
    database_name    = "mydb"
    schema_name      = "public"
    table_name       = "users"
    column_name      = "email"
    masking_function = "anon.fake_email()"
  }
}
`),
				ExpectError: regexp.MustCompile(`Failed to create anonymized branch`),
			},
		},
	})
}
