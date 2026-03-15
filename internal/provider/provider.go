package provider

import (
	"context"
	"net/http"
	"os"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/api_key"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/anonymized_branch"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/branch"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/branches"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/connection_uri"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/data_api"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/database"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/masking_rules"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/neon_auth"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/neon_auth_oauth_provider"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/neon_auth_trusted_domain"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/restore_branch"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/role"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/role_password"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/role_password_reset"
	branch_schema "github.com/kenchan0130/terraform-provider-neon/internal/services/branch/schema"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/set_default_branch"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/branch/snapshot_schedule"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/endpoint/compute_endpoint"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/organization/invitations"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/organization/member"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/organization/org_api_key"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/organization/organization"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/organization/project_transfer"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/organization/vpc_endpoint"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/project/jwks"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/project/project"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/project/project_access"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/project/projects"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/project/recover_project"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/project/restore_snapshot"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/project/snapshot"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/project/vpc_endpoint_restriction"
	"github.com/kenchan0130/terraform-provider-neon/internal/services/region"
)

const defaultBaseURL = "https://console.neon.tech/api/v2"

type NeonProvider struct {
	version    string
	httpClient *http.Client
}

type neonProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NeonProvider{
			version: version,
		}
	}
}

func NewWithHTTPClient(version string, httpClient *http.Client) func() provider.Provider {
	return func() provider.Provider {
		return &NeonProvider{
			version:    version,
			httpClient: httpClient,
		}
	}
}

func (p *NeonProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "neon"
	resp.Version = p.version
}

func (p *NeonProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for Neon serverless Postgres.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "The Neon API key. Can also be set via the `NEON_API_KEY` environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"base_url": schema.StringAttribute{
				Description: "The base URL for the Neon API. Defaults to `https://console.neon.tech/api/v2`.",
				Optional:    true,
			},
		},
	}
}

type neonSecuritySource struct {
	apiKey string
}

func (s *neonSecuritySource) BearerAuth(_ context.Context, _ string) (neon.BearerAuth, error) {
	return neon.BearerAuth{Token: s.apiKey}, nil
}

func (s *neonSecuritySource) CookieAuth(_ context.Context, _ string) (neon.CookieAuth, error) {
	return neon.CookieAuth{}, nil
}

func (s *neonSecuritySource) TokenCookieAuth(_ context.Context, _ string) (neon.TokenCookieAuth, error) {
	return neon.TokenCookieAuth{}, nil
}

func (p *NeonProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config neonProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("NEON_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"The Neon API key must be set via the `api_key` provider attribute or the `NEON_API_KEY` environment variable.",
		)
		return
	}

	baseURL := defaultBaseURL
	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}

	secSource := &neonSecuritySource{apiKey: apiKey}

	var httpClient *http.Client
	if p.httpClient != nil {
		httpClient = p.httpClient
	} else {
		retryClient := retryablehttp.NewClient()
		retryClient.Logger = nil
		retryClient.RequestLogHook = func(_ retryablehttp.Logger, req *http.Request, retryNumber int) {
			tflog.Debug(ctx, "Sending request", map[string]any{
				"method":       req.Method,
				"url":          req.URL.String(),
				"retry_number": retryNumber,
			})
		}
		httpClient = retryClient.StandardClient()
	}

	client, err := neon.NewClient(baseURL, secSource, neon.WithClient(httpClient))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Neon API client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
	resp.ActionData = client
	resp.EphemeralResourceData = client
}

func (p *NeonProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		project.NewResource,
		anonymized_branch.NewResource,
		branch.NewResource,
		compute_endpoint.NewResource,
		database.NewResource,
		role.NewResource,
		vpc_endpoint.NewResource,
		member.NewResource,
		jwks.NewResource,
		project_access.NewResource,
		vpc_endpoint_restriction.NewResource,
		snapshot.NewResource,
		masking_rules.NewResource,
		data_api.NewResource,
		snapshot_schedule.NewResource,
		neon_auth.NewResource,
		neon_auth_oauth_provider.NewResource,
		neon_auth_trusted_domain.NewResource,
		api_key.NewResource,
		org_api_key.NewResource,
	}
}

func (p *NeonProvider) Actions(_ context.Context) []func() action.Action {
	return []func() action.Action{
		project_transfer.NewAction,
		role_password_reset.NewAction,
		set_default_branch.NewAction,
		restore_branch.NewAction,
		restore_snapshot.NewAction,
		recover_project.NewAction,
	}
}

func (p *NeonProvider) EphemeralResources(_ context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		role_password.NewEphemeralResource,
		neon_auth_oauth_provider.NewEphemeralResource,
	}
}

func (p *NeonProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		project.NewDataSource,
		branch.NewDataSource,
		compute_endpoint.NewDataSource,
		database.NewDataSource,
		role.NewDataSource,
		organization.NewDataSource,
		member.NewDataSource,
		vpc_endpoint.NewDataSource,
		jwks.NewDataSource,
		project_access.NewDataSource,
		vpc_endpoint_restriction.NewDataSource,
		anonymized_branch.NewDataSource,
		snapshot.NewDataSource,
		masking_rules.NewDataSource,
		connection_uri.NewDataSource,
		invitations.NewDataSource,
		branch_schema.NewDataSource,
		region.NewDataSource,
		data_api.NewDataSource,
		snapshot_schedule.NewDataSource,
		neon_auth.NewDataSource,
		neon_auth_oauth_provider.NewDataSource,
		role_password.NewDataSource,
		projects.NewDataSource,
		branches.NewDataSource,
	}
}
