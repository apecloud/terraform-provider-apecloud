package utils

import (
	"context"

	"github.com/apecloud/kb-cloud-client-go/api/common"
)

func GetAPICtxAndClient(APIUrl, APIKey, APISecret string, config *common.Configuration, insecureSkipVerify bool) (context.Context, *common.APIClient) {
	if APIKey == "" || APISecret == "" {
		return nil, nil
	}

	withBaseURL := func(ctx context.Context) context.Context {
		return context.WithValue(
			context.WithValue(ctx, common.ContextServerVariables, map[string]string{"site": APIUrl}),
			common.ContextServerIndex,
			0,
		)
	}

	withAuth := func(ctx context.Context) context.Context {
		return context.WithValue(
			ctx,
			common.ContextDigestAuth,
			common.DigestAuth{
				UserName: APIKey,
				Password: APISecret,
			},
		)
	}

	withInsecure := func(ctx context.Context) context.Context {
		return context.WithValue(
			ctx,
			common.ContextInsecureSkipVerify,
			insecureSkipVerify,
		)
	}

	ctx := withAuth(withInsecure(withBaseURL(context.Background())))
	client := common.NewAPIClient(config)
	return ctx, client
}
