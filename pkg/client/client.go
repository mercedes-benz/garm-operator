package client

import (
	"errors"

	"github.com/cloudbase/garm/cmd/garm-cli/client"
	"github.com/cloudbase/garm/cmd/garm-cli/config"
	"github.com/cloudbase/garm/params"
)

type GarmScopeParams struct {
	BaseURL  string
	Username string
	Password string
	Debug    bool
}

// TODO: make managerName configurable
const managerName = "admin"

func newGarmClient(garmParams GarmScopeParams) (*client.Client, error) {
	if garmParams.BaseURL == "" {
		return nil, errors.New("baseURL is mandatory to create a garm client")
	}

	if garmParams.Username == "" {
		return nil, errors.New("username is mandatory to create a garm client")
	}

	if garmParams.Password == "" {
		return nil, errors.New("password is mandator")
	}

	manager := config.Manager{
		Name:    managerName, // TODO: check scope/idea of manager name. ATM it's equal to the username
		BaseURL: garmParams.BaseURL,
	}

	// initiate a new client instance
	garmClient := client.NewClient(manager.Name, manager, garmParams.Debug)

	// login to garm with the operator given credentials to fetch a token
	// for further requests
	token, err := garmClient.Login(garmParams.BaseURL, params.PasswordLoginParams{
		Username: garmParams.Username,
		Password: garmParams.Password,
	})
	if err != nil {
		return nil, err
	}

	garmClient.Config = config.Manager{
		BaseURL: garmParams.BaseURL,
		Token:   token,
		Name:    managerName,
	}
	garmClient.ManagerName = managerName

	// update the client to make use of the generated token
	garmClient = client.NewClient(garmClient.ManagerName, garmClient.Config, garmParams.Debug)

	return garmClient, nil
}
