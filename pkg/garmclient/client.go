package garmclient

import (
	"errors"

	garmClient "github.com/cloudbase/garm/cmd/garm-cli/client"
	garmConfig "github.com/cloudbase/garm/cmd/garm-cli/config"
	garmParams "github.com/cloudbase/garm/params"
)

type GarmClientParams struct {
	BaseURL  string
	Username string
	Password string
}

const managerName = "admin"

func NewGarmClient(params GarmClientParams) (*garmClient.Client, error) {
	if params.BaseURL == "" {
		return nil, errors.New("BaseURL is mandatory to create a garm client")
	}

	// check what's the idea of the manager name - ATM username = managername
	if params.Username == "" {
		return nil, errors.New("Username is mandatory to create a garm client")
	}

	if params.Password == "" {
		return nil, errors.New("Password is mandator")
	}

	manager := garmConfig.Manager{
		Name:    managerName,
		BaseURL: params.BaseURL,
	}

	// initiate the client
	// TODO: could be done only on operator-start?!
	client := garmClient.NewClient(manager.Name, manager, false)

	token, err := client.Login(params.BaseURL, garmParams.PasswordLoginParams{
		Username: params.Username,
		Password: params.Password,
	})
	if err != nil {
		return nil, err
	}

	client.Config = garmConfig.Manager{
		BaseURL: params.BaseURL,
		Token:   token,
		Name:    managerName,
	}
	client.ManagerName = managerName

	client = garmClient.NewClient(client.ManagerName, client.Config, false)

	return client, nil

}
