package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/apecloud/terraform-provider-apecloud/internal/client"
	ds "github.com/apecloud/terraform-provider-apecloud/internal/datasource"
	res "github.com/apecloud/terraform-provider-apecloud/internal/resource"
	"github.com/apecloud/terraform-provider-apecloud/internal/types"
)

type KubeBlockEnterpriseProvider struct {
	// client is the API client to be used by the provider.
	client *client.Client
}

func New() provider.Provider {
	return &KubeBlockEnterpriseProvider{}
}

func (p *KubeBlockEnterpriseProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data types.ProviderSchema
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new API client
	c, err := client.New(data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create API client", err.Error())
		return
	}
	p.client = c

	// Make the client available to all resources
	resp.ResourceData = p.client
	resp.DataSourceData = p.client
}

func (a *KubeBlockEnterpriseProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "apecloud"
}

func (a *KubeBlockEnterpriseProvider) Resources(context.Context) []func() tfresource.Resource {
	return []func() tfresource.Resource{
		res.NewClusterResource,
	}
}

func (a *KubeBlockEnterpriseProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		ds.NewClusterDataSource,
	}
}

func (a *KubeBlockEnterpriseProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "The organization-level API key to use for requests to KubeBlocks Enterprise.",
				Sensitive:   true,
				Optional:    true,
			},
			"api_secret": schema.StringAttribute{
				Description: "The organization-level API secret to use for requests to KubeBlocks Enterprise.",
				Sensitive:   true,
				Optional:    true,
			},
			"admin_api_key": schema.StringAttribute{
				Description: "The admin-level API key to use for requests to KubeBlocks Enterprise.",
				Sensitive:   true,
				Optional:    true,
			},
			"admin_api_secret": schema.StringAttribute{
				Description: "The admin-level API secret to use for requests to KubeBlocks Enterprise.",
				Sensitive:   true,
				Optional:    true,
			},
			"api_url": schema.StringAttribute{
				Description: "The API URL",
				Optional:    true,
			},
			"http_client_retry_enabled": schema.StringAttribute{
				Optional:    true,
				Description: "Enables request retries on HTTP status codes 429 and 5xx. Valid values are [`true`, `false`]. Defaults to `true`.",
			},
			"http_client_retry_timeout": schema.Int64Attribute{
				Optional:    true,
				Description: "The HTTP request retry timeout period. Defaults to 60 seconds.",
			},
			"http_client_retry_backoff_multiplier": schema.Int64Attribute{
				Optional:    true,
				Description: "The HTTP request retry back off multiplier. Defaults to 2.",
			},
			"http_client_retry_backoff_base": schema.Int64Attribute{
				Optional:    true,
				Description: "The HTTP request retry back off base. Defaults to 2.",
			},
			"http_client_retry_max_retries": schema.Int64Attribute{
				Optional:    true,
				Description: "The HTTP request maximum retry number. Defaults to 3.",
			},
			"https_skip_verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Skips TLS verification. Defaults to `false` means not skip TLS verification.",
			},
		},
	}
}

var _ provider.Provider = &KubeBlockEnterpriseProvider{}
