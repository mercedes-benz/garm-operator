package client

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/cloudbase/garm/client"
	apiClientLogin "github.com/cloudbase/garm/client/login"
	"github.com/go-openapi/runtime"
	openapiRuntimeClient "github.com/go-openapi/runtime/client"

	//"github.com/cloudbase/garm/cmd/garm-cli/client"

	"github.com/cloudbase/garm/params"
)

type GarmScopeParams struct {
	BaseURL  string
	Username string
	Password string
	Debug    bool
}

func newGarmClient(garmParams GarmScopeParams) (*client.GarmAPI, runtime.ClientAuthInfoWriter, error) {
	if garmParams.BaseURL == "" {
		return nil, nil, errors.New("baseURL is mandatory to create a garm client")
	}

	if garmParams.Username == "" {
		return nil, nil, errors.New("username is mandatory to create a garm client")
	}

	if garmParams.Password == "" {
		return nil, nil, errors.New("password is mandator")
	}

	baseUrlParsed, err := url.Parse(garmParams.BaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse base url %s: %s", garmParams.BaseURL, err)
	}

	apiPath, err := url.JoinPath(baseUrlParsed.Path, client.DefaultBasePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to join base url path %s with %s: %s", baseUrlParsed.Path, client.DefaultBasePath, err)
	}

	transportCfg := client.DefaultTransportConfig().
		WithHost(baseUrlParsed.Host).
		WithBasePath(apiPath).
		WithSchemes([]string{baseUrlParsed.Scheme})
	apiCli := client.NewHTTPClientWithConfig(nil, transportCfg)
	authToken := openapiRuntimeClient.BearerToken("")

	newLoginParamsReq := apiClientLogin.NewLoginParams()
	newLoginParamsReq.Body = params.PasswordLoginParams{
		Username: garmParams.Username,
		Password: garmParams.Password,
	}

	// login with empty token and login params
	// this will return a new token in response
	resp, err := apiCli.Login.Login(newLoginParamsReq, authToken)
	if err != nil {
		return nil, nil, err
	}

	// update token from login response
	authToken = openapiRuntimeClient.BearerToken(resp.Payload.Token)

	return apiCli, authToken, nil
}
