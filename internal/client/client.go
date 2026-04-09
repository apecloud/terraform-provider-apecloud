package client

import (
	"context"
	"fmt"
	"time"

	"github.com/apecloud/kb-cloud-client-go/api/common"
	"github.com/spf13/cast"

	"github.com/apecloud/terraform-provider-kbcloud/internal/constants"
	"github.com/apecloud/terraform-provider-kbcloud/internal/types"
	"github.com/apecloud/terraform-provider-kbcloud/internal/utils"
)

// Client is the API client for the ApeCloud API.

type Client struct {
	cli      *common.APIClient
	adminCli *common.APIClient
	ctx      context.Context
	adminCtx context.Context
}

func (c *Client) IsAdminClient() bool {
	return c.adminCli != nil
}

func (c *Client) AdminClient() *common.APIClient {
	return c.adminCli
}

func (c *Client) AdminCtx() context.Context {
	return c.adminCtx
}

func (c *Client) Ctx() context.Context {
	return c.ctx
}

func (c *Client) Client() *common.APIClient {
	return c.cli
}

// New creates a new API client.
func New(data types.ProviderSchema) (*Client, error) {
	var (
		apiURL string = data.APIUrl.ValueString()
		err    error
	)
	if data.APIUrl.IsNull() || data.APIUrl.IsUnknown() {
		apiURL, err = utils.GetMultiEnvVar(constants.EnvVarAPIUrl)
		if err != nil {
			return nil, fmt.Errorf("api_url or env var %s must be set", constants.EnvVarAPIUrl)
		}
	}

	config := common.NewConfiguration()
	if cast.ToBool(data.HTTPClientRetryEnabled.ValueString()) {
		config.RetryConfiguration.EnableRetry = true
		config.RetryConfiguration.MaxRetries = cast.ToInt(data.HTTPClientRetryMaxRetries.ValueInt64())
		config.RetryConfiguration.HTTPRetryTimeout = time.Duration(data.HTTPClientRetryTimeout.ValueInt64()) * time.Second
	}

	if !data.HTTPClientRetryBackoffMultiplier.IsNull() && !data.HTTPClientRetryBackoffMultiplier.IsUnknown() {
		config.RetryConfiguration.BackOffMultiplier = cast.ToFloat64(data.HTTPClientRetryBackoffMultiplier.ValueInt64())
	}

	if !data.HTTPClientRetryBackoffBase.IsNull() && !data.HTTPClientRetryBackoffBase.IsUnknown() {
		config.RetryConfiguration.BackOffBase = cast.ToFloat64(data.HTTPClientRetryBackoffBase.ValueInt64())
	}

	ctx, cli := utils.GetAPICtxAndClient(apiURL, data.APIKey.ValueString(), data.APISecret.ValueString(), config, data.HTTPSSkipVerify.ValueBool())
	adminCtx, adminCli := utils.GetAPICtxAndClient(apiURL, data.AdminAPIKey.ValueString(), data.AdminAPISecret.ValueString(), config, data.HTTPSSkipVerify.ValueBool())

	if cli == nil && adminCli == nil {
		return nil, fmt.Errorf("must provide api_key/api_secret or admin_api_key/admin_api_secret or both")
	}

	return &Client{
		cli:      cli,
		adminCli: adminCli,
		ctx:      ctx,
		adminCtx: adminCtx,
	}, nil
}
