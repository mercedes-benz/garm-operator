// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	garm "github.com/cloudbase/garm/client"
	apiClientFirstRun "github.com/cloudbase/garm/client/first_run"
	apiClientLogin "github.com/cloudbase/garm/client/login"
	"github.com/cloudbase/garm/params"
	"github.com/go-openapi/runtime"
	openapiRuntimeClient "github.com/go-openapi/runtime/client"
	"github.com/golang-jwt/jwt/v4"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

var Client GarmClient

type GarmScopeParams struct {
	BaseURL  string
	Username string
	Password string
	Debug    bool
	Email    string
}

type GarmClient interface {
	GarmAPI() *garm.GarmAPI
	Token() runtime.ClientAuthInfoWriter
	Login() error
	Init() error
}

type garmClient struct {
	client     *garm.GarmAPI
	token      runtime.ClientAuthInfoWriter
	garmParams GarmScopeParams
}

func (s *garmClient) GarmAPI() *garm.GarmAPI {
	return s.client
}

func (s *garmClient) Token() runtime.ClientAuthInfoWriter {
	return s.token
}

func (s *garmClient) Login() error {
	metrics.TotalGarmCalls.WithLabelValues("Login").Inc()
	authenticatedClient, authInfoWriter, err := newGarmClient(s.garmParams)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("Login").Inc()
		return err
	}
	s.client = authenticatedClient
	s.token = authInfoWriter

	return nil
}

func (s *garmClient) Init() error {
	ctx := context.Background()
	metrics.TotalGarmCalls.WithLabelValues("Init").Inc()
	err := initializeGarm(ctx, s.garmParams)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("Init").Inc()
		return err
	}

	return nil
}

func CreateInstance(garmParams GarmScopeParams) error {
	Client = &garmClient{
		garmParams: garmParams,
	}
	if err := Client.Init(); err != nil {
		return fmt.Errorf("failed to initialize GARM: %w", err)
	}
	if err := Client.Login(); err != nil {
		return fmt.Errorf("failed to login to garm client: %w", err)
	}
	return nil
}

func newGarmClient(garmParams GarmScopeParams) (*garm.GarmAPI, runtime.ClientAuthInfoWriter, error) {
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

	apiPath, err := url.JoinPath(baseURLParsed.Path, garm.DefaultBasePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to join base url path %s with %s: %s", baseURLParsed.Path, garm.DefaultBasePath, err)
	}

	transportCfg := garm.DefaultTransportConfig().
		WithHost(baseURLParsed.Host).
		WithBasePath(apiPath).
		WithSchemes([]string{baseURLParsed.Scheme})
	apiCli := garm.NewHTTPClientWithConfig(nil, transportCfg)
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
	extractJWTTokenExp(context.TODO(), resp.Payload.Token)
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

	apiPath, err := url.JoinPath(baseURLParsed.Path, garm.DefaultBasePath)
	if err != nil {
		return fmt.Errorf("failed to join base url path %s with %s: %s", baseURLParsed.Path, garm.DefaultBasePath, err)
	}

	transportCfg := garm.DefaultTransportConfig().
		WithHost(baseURLParsed.Host).
		WithBasePath(apiPath).
		WithSchemes([]string{baseURLParsed.Scheme})
	apiCli := garm.NewHTTPClientWithConfig(nil, transportCfg)
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

func IsUnauthenticatedError(err interface{}) bool {
	apiErr, ok := err.(runtime.ClientResponseStatus)
	if !ok {
		return false
	}
	return apiErr.IsCode(401)
}

func IsNotFoundError(err interface{}) bool {
	apiErr, ok := err.(runtime.ClientResponseStatus)
	if !ok {
		return false
	}
	return apiErr.IsCode(404)
}

type Func[T interface{}] func() (T, error)

func EnsureAuth[T interface{}](f Func[T]) (T, error) {
	result, err := f()
	if err != nil && IsUnauthenticatedError(err) {
		metrics.GarmCallErrors.WithLabelValues("client.Unauthenticated").Inc()

		err = Client.Init()
		if err != nil {
			return result, err
		}

		err = Client.Login()
		if err != nil {
			return result, err
		}

		result, err = f()
	}
	return result, err
}

func extractJWTTokenExp(ctx context.Context, tokenString string) {
	log := log.FromContext(ctx)

	log.Info("Extracting expiry date of jwt")
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		log.Error(err, "failed parsing jwt")
		return
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if exp, ok := claims["exp"].(float64); ok {
			expTime := time.Unix(int64(exp), 0)
			log.Info(fmt.Sprintf("new token expires on " + expTime.Format(time.UnixDate)))
			metrics.GarmJwtExpiresAt.Set(exp)
		}
	}
}
