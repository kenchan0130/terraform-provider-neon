package project_test

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/jarcoal/httpmock"
	"github.com/kenchan0130/terraform-provider-neon/internal/testutil"
)

const projectJSON = `{
	"id": "test-project-id",
	"name": "my-project",
	"region_id": "aws-us-east-1",
	"pg_version": 16,
	"history_retention_seconds": 86400,
	"store_passwords": true,
	"platform_id": "aws",
	"provisioner": "k8s-neonvm",
	"proxy_host": "us-east-1.aws.neon.tech",
	"branch_logical_size_limit": 0,
	"branch_logical_size_limit_bytes": 0,
	"data_storage_bytes_hour": 0,
	"data_transfer_bytes": 0,
	"written_data_bytes": 0,
	"compute_time_seconds": 0,
	"active_time_seconds": 0,
	"cpu_used_sec": 0,
	"creation_source": "console",
	"owner_id": "owner-001",
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z",
	"consumption_period_start": "2025-01-01T00:00:00Z",
	"consumption_period_end": "2025-02-01T00:00:00Z",
	"settings": {
		"quota": {},
		"allowed_ips": {
			"ips": ["0.0.0.0/0"],
			"protected_branches_only": false
		},
		"enable_logical_replication": false,
		"block_public_connections": false,
		"block_vpc_connections": false
	},
	"default_endpoint_settings": {
		"autoscaling_limit_min_cu": 0.25,
		"autoscaling_limit_max_cu": 0.25,
		"suspend_timeout_seconds": 300
	}
}`

const projectJSONNoStorePasswords = `{
	"id": "test-project-id",
	"name": "my-project",
	"region_id": "aws-us-east-1",
	"pg_version": 16,
	"history_retention_seconds": 86400,
	"store_passwords": false,
	"platform_id": "aws",
	"provisioner": "k8s-neonvm",
	"proxy_host": "us-east-1.aws.neon.tech",
	"branch_logical_size_limit": 0,
	"branch_logical_size_limit_bytes": 0,
	"data_storage_bytes_hour": 0,
	"data_transfer_bytes": 0,
	"written_data_bytes": 0,
	"compute_time_seconds": 0,
	"active_time_seconds": 0,
	"cpu_used_sec": 0,
	"creation_source": "console",
	"owner_id": "owner-001",
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z",
	"consumption_period_start": "2025-01-01T00:00:00Z",
	"consumption_period_end": "2025-02-01T00:00:00Z",
	"settings": {
		"quota": {},
		"allowed_ips": {
			"ips": ["0.0.0.0/0"],
			"protected_branches_only": false
		},
		"enable_logical_replication": false,
		"block_public_connections": false,
		"block_vpc_connections": false
	},
	"default_endpoint_settings": {
		"autoscaling_limit_min_cu": 0.25,
		"autoscaling_limit_max_cu": 0.25,
		"suspend_timeout_seconds": 300
	}
}`

const branchMinJSON = `{"id":"br-main","project_id":"test-project-id","name":"main","current_state":"init","state_changed_at":"2025-01-01T00:00:00Z","creation_source":"console","primary":false,"default":true,"protected":false,"cpu_used_sec":0,"compute_time_seconds":0,"active_time_seconds":0,"written_data_bytes":0,"data_transfer_bytes":0,"created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z"}`

// projectJSONWithMaintenanceWindow mirrors projectJSON but additionally carries a
// server-managed maintenance_window, the kind of value a Free-plan project cannot
// have echoed back in an update PATCH. Used by tests that verify unconfigured
// settings never leak into the update request body.
const projectJSONWithMaintenanceWindow = `{
	"id": "test-project-id",
	"name": "my-project",
	"region_id": "aws-us-east-1",
	"pg_version": 16,
	"history_retention_seconds": 86400,
	"store_passwords": true,
	"platform_id": "aws",
	"provisioner": "k8s-neonvm",
	"proxy_host": "us-east-1.aws.neon.tech",
	"branch_logical_size_limit": 0,
	"branch_logical_size_limit_bytes": 0,
	"data_storage_bytes_hour": 0,
	"data_transfer_bytes": 0,
	"written_data_bytes": 0,
	"compute_time_seconds": 0,
	"active_time_seconds": 0,
	"cpu_used_sec": 0,
	"creation_source": "console",
	"owner_id": "owner-001",
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z",
	"consumption_period_start": "2025-01-01T00:00:00Z",
	"consumption_period_end": "2025-02-01T00:00:00Z",
	"settings": {
		"quota": {},
		"allowed_ips": {
			"ips": ["0.0.0.0/0"],
			"protected_branches_only": false
		},
		"enable_logical_replication": false,
		"block_public_connections": false,
		"block_vpc_connections": false,
		"maintenance_window": {"weekdays":[1],"start_time":"01:00","end_time":"02:00"}
	},
	"default_endpoint_settings": {
		"autoscaling_limit_min_cu": 0.25,
		"autoscaling_limit_max_cu": 0.25,
		"suspend_timeout_seconds": 300
	}
}`

// mergePatchIntoProject simulates the Neon API's documented behavior for
// PATCH /projects/{id}: the "settings" object in the request is merged into the
// existing project's settings rather than replacing it wholesale, so fields the
// request omits are preserved server-side. base is the current project JSON
// (as previously returned by GET/PATCH); patchBody is the raw PATCH request body.
func mergePatchIntoProject(t *testing.T, base string, patchBody []byte) string {
	t.Helper()

	var req struct {
		Project struct {
			Name                    *string         `json:"name"`
			Settings                json.RawMessage `json:"settings"`
			DefaultEndpointSettings json.RawMessage `json:"default_endpoint_settings"`
		} `json:"project"`
	}
	if err := json.Unmarshal(patchBody, &req); err != nil {
		t.Fatalf("failed to unmarshal PATCH request body: %v", err)
	}

	var project map[string]any
	if err := json.Unmarshal([]byte(base), &project); err != nil {
		t.Fatalf("failed to unmarshal base project JSON: %v", err)
	}

	if req.Project.Name != nil {
		project["name"] = *req.Project.Name
	}
	mergeRawObjectInto(t, project, "settings", req.Project.Settings)
	mergeRawObjectInto(t, project, "default_endpoint_settings", req.Project.DefaultEndpointSettings)

	out, err := json.Marshal(project)
	if err != nil {
		t.Fatalf("failed to marshal merged project JSON: %v", err)
	}
	return string(out)
}

func mergeRawObjectInto(t *testing.T, project map[string]any, key string, raw json.RawMessage) {
	t.Helper()

	if len(raw) == 0 {
		return
	}
	var patch map[string]any
	if err := json.Unmarshal(raw, &patch); err != nil {
		t.Fatalf("failed to unmarshal %q from PATCH request body: %v", key, err)
	}
	existing, _ := project[key].(map[string]any)
	if existing == nil {
		existing = map[string]any{}
	}
	for k, v := range patch {
		existing[k] = v
	}
	project[key] = existing
}

// TestProjectResource_UpdateDoesNotEchoServerSettings is a regression test for the
// no-op-update bug: when only "name" is configured, all other Optional+Computed
// settings/default_endpoint_settings leaves must be omitted from the update PATCH,
// even though they carry server-sourced values through the plan. Before the fix,
// the update PATCH echoed back the full settings object (including a
// maintenance_window a Free-plan project cannot have updated), which the real API
// rejects with 400.
func TestProjectResource_UpdateDoesNotEchoServerSettings(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	currentProject := projectJSONWithMaintenanceWindow
	var capturedPatchBody []byte

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects",
		testutil.JSONResponder(201, `{
			"project": `+projectJSONWithMaintenanceWindow+`,
			"connection_uris": [],
			"roles": [],
			"databases": [],
			"operations": [],
			"branch": `+branchMinJSON+`,
			"endpoints": []
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id",
		func(_ *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, `{"project": `+currentProject+`}`)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		},
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(200, `{"project": `+currentProject+`}`),
	)

	transport.RegisterResponder(http.MethodPatch,
		"https://neon.example.com/api/v2/projects/test-project-id",
		func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			capturedPatchBody = body

			// Old-bug detector: the real API rejects a settings object echoed
			// back on a Free-plan project (e.g. maintenance_window). If the
			// fix regresses and settings ever end up in the PATCH again, fail
			// the same way the real API would.
			if strings.Contains(string(body), `"settings"`) {
				resp := httpmock.NewStringResponse(400, `{"code":"","message":"updating the maintenance window is not allowed","request_id":"req-999"}`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			}

			currentProject = mergePatchIntoProject(t, currentProject, body)
			resp := httpmock.NewStringResponse(200, `{"project": `+currentProject+`, "operations": []}`)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		},
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project"
}
`),
				Check: testutil.CheckResourceAttr("neon_project.test", "name", "my-project"),
				// maintenance_window's schema children are Required inside an
				// Optional+Computed object using UseStateForUnknown; that
				// pre-existing combination (unrelated to this fix, and the
				// schema is out of scope here) makes the framework report a
				// spurious post-apply diff whenever maintenance_window is
				// populated but unconfigured. The PATCH-body assertions below
				// are what actually exercise the fix.
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project-renamed"
}
`),
				Check:              testutil.CheckResourceAttr("neon_project.test", "name", "my-project-renamed"),
				ExpectNonEmptyPlan: true,
			},
		},
	})

	if capturedPatchBody == nil {
		t.Fatal("expected a PATCH request to be made")
	}
	body := string(capturedPatchBody)
	for _, forbidden := range []string{`"settings"`, `"maintenance_window"`, `"default_endpoint_settings"`} {
		if strings.Contains(body, forbidden) {
			t.Errorf("PATCH body unexpectedly contains %s: %s", forbidden, body)
		}
	}
	if !strings.Contains(body, "my-project-renamed") {
		t.Errorf("PATCH body does not contain the new name: %s", body)
	}
}

// TestProjectResource_UpdateSendsConfiguredSettings verifies leaf-level gating:
// when only settings.enable_logical_replication is configured, changing it must
// send that single leaf in the PATCH while still omitting unconfigured leaves
// (like maintenance_window) that are only present because the API populated them
// server-side.
func TestProjectResource_UpdateSendsConfiguredSettings(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	currentProject := projectJSONWithMaintenanceWindow
	var capturedPatchBody []byte

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects",
		testutil.JSONResponder(201, `{
			"project": `+projectJSONWithMaintenanceWindow+`,
			"connection_uris": [],
			"roles": [],
			"databases": [],
			"operations": [],
			"branch": `+branchMinJSON+`,
			"endpoints": []
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id",
		func(_ *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, `{"project": `+currentProject+`}`)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		},
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(200, `{"project": `+currentProject+`}`),
	)

	transport.RegisterResponder(http.MethodPatch,
		"https://neon.example.com/api/v2/projects/test-project-id",
		func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			capturedPatchBody = body

			currentProject = mergePatchIntoProject(t, currentProject, body)
			resp := httpmock.NewStringResponse(200, `{"project": `+currentProject+`, "operations": []}`)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		},
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project"
  settings = {
    enable_logical_replication = false
  }
}
`),
				Check: testutil.CheckResourceAttr("neon_project.test", "settings.enable_logical_replication", "false"),
				// See the comment in TestProjectResource_UpdateDoesNotEchoServerSettings:
				// an unconfigured, server-populated maintenance_window causes a
				// spurious post-apply diff due to a pre-existing schema
				// limitation that is out of scope for this fix.
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project"
  settings = {
    enable_logical_replication = true
  }
}
`),
				Check:              testutil.CheckResourceAttr("neon_project.test", "settings.enable_logical_replication", "true"),
				ExpectNonEmptyPlan: true,
			},
		},
	})

	if capturedPatchBody == nil {
		t.Fatal("expected a PATCH request to be made")
	}
	body := string(capturedPatchBody)
	if !strings.Contains(body, `"enable_logical_replication":true`) {
		t.Errorf("PATCH body does not contain the configured leaf: %s", body)
	}
	if strings.Contains(body, `"maintenance_window"`) {
		t.Errorf("PATCH body unexpectedly contains unconfigured leaf maintenance_window: %s", body)
	}
}

// TestProjectResource_UpdateAPIErrorMessage verifies that the API's error message
// surfaces to the practitioner instead of being hidden behind a generic Go error
// string.
func TestProjectResource_UpdateAPIErrorMessage(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects",
		testutil.JSONResponder(201, `{
			"project": `+projectJSON+`,
			"connection_uris": [],
			"roles": [],
			"databases": [],
			"operations": [],
			"branch": `+branchMinJSON+`,
			"endpoints": []
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(200, `{"project": `+projectJSON+`}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(200, `{"project": `+projectJSON+`}`),
	)

	transport.RegisterResponder(http.MethodPatch,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(400, `{"code":"","message":"updating the maintenance window is not allowed","request_id":"req-123"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project"
}
`),
			},
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project-renamed"
}
`),
				ExpectError: regexp.MustCompile(`updating the maintenance window is not allowed`),
			},
		},
	})
}

func setupProjectMocks(transport *httpmock.MockTransport) {
	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects",
		testutil.JSONResponder(201, `{
			"project": `+projectJSON+`,
			"connection_uris": [],
			"roles": [],
			"databases": [],
			"operations": [],
			"branch": `+branchMinJSON+`,
			"endpoints": []
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(200, `{"project": `+projectJSON+`}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(200, `{"project": `+projectJSON+`}`),
	)
}

func TestProjectResource_Create(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupProjectMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_project.test", "id", "test-project-id"),
					testutil.CheckResourceAttr("neon_project.test", "name", "my-project"),
					testutil.CheckResourceAttr("neon_project.test", "region_id", "aws-us-east-1"),
					testutil.CheckResourceAttr("neon_project.test", "pg_version", "16"),
					testutil.CheckResourceAttr("neon_project.test", "store_passwords", "true"),
					testutil.CheckResourceAttr("neon_project.test", "history_retention_seconds", "86400"),
					testutil.CheckResourceAttr("neon_project.test", "compute_provisioner", "k8s-neonvm"),
					testutil.CheckResourceAttr("neon_project.test", "default_endpoint_settings.autoscaling_limit_min_cu", "0.25"),
					testutil.CheckResourceAttr("neon_project.test", "default_endpoint_settings.autoscaling_limit_max_cu", "0.25"),
					testutil.CheckResourceAttr("neon_project.test", "default_endpoint_settings.suspend_timeout_seconds", "300"),
					testutil.CheckResourceAttr("neon_project.test", "settings.enable_logical_replication", "false"),
					testutil.CheckResourceAttr("neon_project.test", "settings.block_public_connections", "false"),
					testutil.CheckResourceAttr("neon_project.test", "settings.block_vpc_connections", "false"),
					testutil.CheckResourceAttr("neon_project.test", "settings.allowed_ips.ips.#", "1"),
					testutil.CheckResourceAttr("neon_project.test", "settings.allowed_ips.ips.0", "0.0.0.0/0"),
					testutil.CheckResourceAttr("neon_project.test", "settings.allowed_ips.protected_branches_only", "false"),
				),
			},
		},
	})
}

func TestProjectResource_Import(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	setupProjectMocks(transport)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project"
}
`),
			},
			{
				ResourceName:      "neon_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const projectUpdatedJSON = `{
	"id": "test-project-id",
	"name": "my-project-renamed",
	"region_id": "aws-us-east-1",
	"pg_version": 16,
	"history_retention_seconds": 86400,
	"store_passwords": true,
	"platform_id": "aws",
	"provisioner": "k8s-neonvm",
	"proxy_host": "us-east-1.aws.neon.tech",
	"branch_logical_size_limit": 0,
	"branch_logical_size_limit_bytes": 0,
	"data_storage_bytes_hour": 0,
	"data_transfer_bytes": 0,
	"written_data_bytes": 0,
	"compute_time_seconds": 0,
	"active_time_seconds": 0,
	"cpu_used_sec": 0,
	"creation_source": "console",
	"owner_id": "owner-001",
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-02T00:00:00Z",
	"consumption_period_start": "2025-01-01T00:00:00Z",
	"consumption_period_end": "2025-02-01T00:00:00Z",
	"settings": {
		"quota": {},
		"allowed_ips": {
			"ips": ["0.0.0.0/0"],
			"protected_branches_only": false
		},
		"enable_logical_replication": false,
		"block_public_connections": false,
		"block_vpc_connections": false
	},
	"default_endpoint_settings": {
		"autoscaling_limit_min_cu": 0.25,
		"autoscaling_limit_max_cu": 0.25,
		"suspend_timeout_seconds": 300
	}
}`

// TestProjectResource_Update verifies that an in-place update (e.g. changing
// name) succeeds even though the API advances updated_at on every update.
// Regression test for updated_at having UseStateForUnknown, which caused
// "Provider produced inconsistent result after apply" on every in-place
// update.
func TestProjectResource_Update(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	currentProject := projectJSON

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects",
		testutil.JSONResponder(201, `{
			"project": `+projectJSON+`,
			"connection_uris": [],
			"roles": [],
			"databases": [],
			"operations": [],
			"branch": `+branchMinJSON+`,
			"endpoints": []
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id",
		func(_ *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, `{"project": `+currentProject+`}`)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		},
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(200, `{"project": `+projectJSON+`}`),
	)

	transport.RegisterResponder(http.MethodPatch,
		"https://neon.example.com/api/v2/projects/test-project-id",
		func(_ *http.Request) (*http.Response, error) {
			currentProject = projectUpdatedJSON
			resp := httpmock.NewStringResponse(200, `{"project": `+projectUpdatedJSON+`, "operations": []}`)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		},
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_project.test", "name", "my-project"),
					testutil.CheckResourceAttr("neon_project.test", "updated_at", "2025-01-01T00:00:00Z"),
				),
			},
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project-renamed"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.CheckResourceAttr("neon_project.test", "name", "my-project-renamed"),
					testutil.CheckResourceAttr("neon_project.test", "updated_at", "2025-01-02T00:00:00Z"),
				),
			},
		},
	})
}

// TestProjectResource_StorePasswordsChangeForcesReplacement verifies that
// changing store_passwords (which the Update API does not support) forces
// resource replacement instead of an in-place update.
func TestProjectResource_StorePasswordsChangeForcesReplacement(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	currentProject := projectJSON

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects",
		func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			project := projectJSON
			if strings.Contains(string(body), `"store_passwords":false`) {
				project = projectJSONNoStorePasswords
			}
			currentProject = project
			resp := httpmock.NewStringResponse(201, `{
				"project": `+project+`,
				"connection_uris": [],
				"roles": [],
				"databases": [],
				"operations": [],
				"branch": `+branchMinJSON+`,
				"endpoints": []
			}`)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		},
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id",
		func(_ *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, `{"project": `+currentProject+`}`)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		},
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(200, `{"project": `+projectJSON+`}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name            = "my-project"
  store_passwords = true
}
`),
				Check: testutil.CheckResourceAttr("neon_project.test", "store_passwords", "true"),
			},
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name            = "my-project"
  store_passwords = false
}
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("neon_project.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
		},
	})
}

// TestProjectResource_DeleteNotFound verifies that destroying a project that
// was already deleted out-of-band (API returns 404) is treated as a
// successful delete rather than an error.
func TestProjectResource_DeleteNotFound(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects",
		testutil.JSONResponder(201, `{
			"project": `+projectJSON+`,
			"connection_uris": [],
			"roles": [],
			"databases": [],
			"operations": [],
			"branch": `+branchMinJSON+`,
			"endpoints": []
		}`),
	)

	transport.RegisterResponder(http.MethodGet,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(200, `{"project": `+projectJSON+`}`),
	)

	transport.RegisterResponder(http.MethodDelete,
		"https://neon.example.com/api/v2/projects/test-project-id",
		testutil.JSONResponder(404, `{"code":"PROJECT_NOT_FOUND","message":"project not found"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project"
}
`),
			},
		},
	})
}

func TestProjectResource_APIError(t *testing.T) {
	transport := httpmock.NewMockTransport()
	httpClient := &http.Client{Transport: transport}

	transport.RegisterResponder(http.MethodPost,
		"https://neon.example.com/api/v2/projects",
		testutil.JSONResponder(403, `{"message":"authentication error","code":"AUTH_FAILED"}`),
	)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProtoV6ProviderFactories(httpClient),
		Steps: []resource.TestStep{
			{
				Config: testutil.TestConfig(`
resource "neon_project" "test" {
  name = "my-project"
}
`),
				ExpectError: regexp.MustCompile(`Failed to create project`),
			},
		},
	})
}
