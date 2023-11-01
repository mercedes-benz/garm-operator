// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/cloudbase/garm/client"
	apiClientFirstRun "github.com/cloudbase/garm/client/first_run"
	apiClientLogin "github.com/cloudbase/garm/client/login"
	"github.com/cloudbase/garm/params"
	"github.com/go-openapi/runtime"
	openapiRuntimeClient "github.com/go-openapi/runtime/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

type GarmScopeParams struct {
	BaseURL  string
	Username string
	Password string
	Debug    bool
	Email    string
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

	baseURLParsed, err := url.Parse(garmParams.BaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse base url %s: %s", garmParams.BaseURL, err)
	}

	apiPath, err := url.JoinPath(baseURLParsed.Path, client.DefaultBasePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to join base url path %s with %s: %s", baseURLParsed.Path, client.DefaultBasePath, err)
	}

	transportCfg := client.DefaultTransportConfig().
		WithHost(baseURLParsed.Host).
		WithBasePath(apiPath).
		WithSchemes([]string{baseURLParsed.Scheme})
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
	metrics.TotalGarmCalls.WithLabelValues("client.Login").Inc()
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("client.Login").Inc()
		return nil, nil, err
	}

	// update token from login response
	authToken = openapiRuntimeClient.BearerToken(resp.Payload.Token)

	return apiCli, authToken, nil
}

func initializeGarm(ctx context.Context, garmParams GarmScopeParams) error {
	log := log.FromContext(ctx)

	newUserReq := apiClientFirstRun.NewFirstRunParams()
	newUserReq.Body = params.NewUserParams{
		Username: garmParams.Username,
		Password: garmParams.Password,
		Email:    garmParams.Email,
	}

	baseURLParsed, err := url.Parse(garmParams.BaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse base url %s: %s", garmParams.BaseURL, err)
	}

	apiPath, err := url.JoinPath(baseURLParsed.Path, client.DefaultBasePath)
	if err != nil {
		return fmt.Errorf("failed to join base url path %s with %s: %s", baseURLParsed.Path, client.DefaultBasePath, err)
	}

	transportCfg := client.DefaultTransportConfig().
		WithHost(baseURLParsed.Host).
		WithBasePath(apiPath).
		WithSchemes([]string{baseURLParsed.Scheme})
	apiCli := client.NewHTTPClientWithConfig(nil, transportCfg)
	authToken := openapiRuntimeClient.BearerToken("")

	resp, err := apiCli.FirstRun.FirstRun(newUserReq, authToken)
	if err != nil {
		if strings.Contains(err.Error(), "(status 409)") {
			log.Info("Garm is already initialized")
			return nil
		}
		return fmt.Errorf("failed to initialize garm: %s", err)
	}

	log.Info("Garm initialized successfully with the following User", "ID", resp.Payload.ID, "username", resp.Payload.Username, "email", resp.Payload.Email, "enabled", resp.Payload.Enabled)

	return nil
}

func IsNotFoundError(err interface{}) bool {
	apiErr, ok := err.(runtime.ClientResponseStatus)
	if !ok {
		return false
	}
	return apiErr.IsCode(404)
}
