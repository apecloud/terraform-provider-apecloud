package types

import "github.com/hashicorp/terraform-plugin-framework/types"

// ProviderSchema defines the schema for the provider.
type ProviderSchema struct {
	APIKey    types.String `tfsdk:"api_key"`
	APISecret types.String `tfsdk:"api_secret"`

	AdminAPIKey    types.String `tfsdk:"admin_api_key"`
	AdminAPISecret types.String `tfsdk:"admin_api_secret"`
	APIUrl         types.String `tfsdk:"api_url"`

	HTTPClientRetryEnabled           types.String `tfsdk:"http_client_retry_enabled"`
	HTTPClientRetryTimeout           types.Int64  `tfsdk:"http_client_retry_timeout"`
	HTTPClientRetryBackoffMultiplier types.Int64  `tfsdk:"http_client_retry_backoff_multiplier"`
	HTTPClientRetryBackoffBase       types.Int64  `tfsdk:"http_client_retry_backoff_base"`
	HTTPClientRetryMaxRetries        types.Int64  `tfsdk:"http_client_retry_max_retries"`
	HTTPSSkipVerify                  types.Bool   `tfsdk:"https_skip_verify"`
}
