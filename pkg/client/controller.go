// SPDX-License-Identifier: MIT
package client

import (
	"github.com/cloudbase/garm/client/controller"
	garmcontroller "github.com/cloudbase/garm/client/controller"
	"github.com/cloudbase/garm/client/controller_info"
	"github.com/cloudbase/garm/params"

	"github.com/mercedes-benz/garm-operator/pkg/metrics"
	"github.com/mercedes-benz/garm-operator/pkg/util"
)

type ControllerClient interface {
	GetControllerInfo() (*controller_info.ControllerInfoOK, error)
	UpdateController(params *controller.UpdateControllerParams) (*controller.UpdateControllerOK, error)
}

type controllerClient struct {
	GarmClient
}

func NewControllerClient() ControllerClient {
	return &controllerClient{
		Client,
	}
}

const (
	initialMetadataURL = "https://initial.metadata.garm.local"
	initialCallbackURL = "https://initial.callback.garm.local"
	initialWebhookURL  = "https://initial.webhook.garm.local"
)

func (s *controllerClient) GetControllerInfo() (*controller_info.ControllerInfoOK, error) {
	return EnsureAuth(func() (*controller_info.ControllerInfoOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("controller.Info").Inc()
		controllerInfo, err := s.GarmAPI().ControllerInfo.ControllerInfo(&controller_info.ControllerInfoParams{}, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("controller.Info").Inc()

			// after the first run, garm needs a configuration for webhook, metadata and callback
			// to make garm work after the first run, we set some defaults
			if IsConflictError(err) {
				updateParams := garmcontroller.NewUpdateControllerParams().WithBody(params.UpdateControllerParams{
					MetadataURL: util.StringPtr(initialMetadataURL),
					CallbackURL: util.StringPtr(initialCallbackURL),
					WebhookURL:  util.StringPtr(initialWebhookURL),
				})
				// let's initiate the new controller with some defaults
				_, err := s.UpdateController(updateParams)
				if err != nil {
					return nil, err
				}
			}
			return nil, err
		}
		return controllerInfo, nil
	})
}

func (s *controllerClient) UpdateController(param *controller.UpdateControllerParams) (*controller.UpdateControllerOK, error) {
	return EnsureAuth(func() (*controller.UpdateControllerOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("controller.Update").Inc()
		enterprise, err := s.GarmAPI().Controller.UpdateController(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("controller.Update").Inc()
			return nil, err
		}
		return enterprise, nil
	})
}
