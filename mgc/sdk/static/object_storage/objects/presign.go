package objects

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"magalu.cloud/core"
	mgcAuthPkg "magalu.cloud/core/auth"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

type presignObjectParams struct {
	Destination mgcSchemaPkg.URI `json:"dst" jsonschema:"description=Path of the object to generate pre-signed URL for,example=bucket1/file.txt" mgc:"positional"`
	Expiry      string           `json:"expires-in" jsonschema_description:"Expiration time for the pre-signed URL. Valid time units are 'ns, 'us' (or 'µs'), 'ms', 's',  'm', and 'h'." jsonschema:"example=2h"`
	Method      string           `json:"method" jsonschema:"enum=GET,enum=PUT,default=GET"`
}

type presignedUrlResult struct {
	URL mgcSchemaPkg.URI `json:"url"`
}

var getPresign = utils.NewLazyLoader[core.Executor](func() core.Executor {
	executor := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "presign",
			Description: "Generate a pre-signed URL for accessing an object",
		},
		presign,
	)
	return core.NewExecuteResultOutputOptions(executor, func(exec core.Executor, result core.Result) string {
		return "template={{.url}}\n"
	})
})

func presign(ctx context.Context, p presignObjectParams, cfg common.Config) (presignResult *presignedUrlResult, err error) {
	req, err := newPresignedRequest(ctx, cfg, p)
	if err != nil {
		return
	}

	auth := mgcAuthPkg.FromContext(ctx)
	if auth == nil {
		return nil, fmt.Errorf("programming error: unable to get auth from context")
	}

	accessKey, accessSecretKey := auth.AccessKeyPair()

	expirationTime, err := time.ParseDuration(p.Expiry)
	if err != nil {
		return nil, core.UsageError{Err: fmt.Errorf("error when parsing the expirationTime for presigned url: %w", err)}
	}

	presignedURL, err := getPresignedURL(req, accessKey, accessSecretKey, expirationTime)
	if err != nil {
		return
	}
	return &presignedUrlResult{
		URL: mgcSchemaPkg.URI(presignedURL),
	}, nil
}

func newPresignedRequest(ctx context.Context, cfg common.Config, p presignObjectParams) (*http.Request, error) {
	host, err := common.BuildBucketHostWithPath(cfg, common.NewBucketNameFromURI(p.Destination), p.Destination.Path())
	if err != nil {
		return nil, core.UsageError{Err: err}
	}
	return http.NewRequestWithContext(ctx, p.Method, string(host), nil)
}

func getPresignedURL(req *http.Request, accessKey, secretKey string, expirationTime time.Duration) (presignedUrl string, err error) {
	if expirationTime < time.Second || expirationTime > 604000*time.Second {
		err = core.UsageError{Err: fmt.Errorf("expirationTime for presigned URL should be between 1 second and 7 days")}
		return
	}

	url, err := common.SignedUrl(req, accessKey, secretKey, expirationTime)
	if err != nil {
		return
	}
	return url.String(), nil
}
