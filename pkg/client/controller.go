package client

import (
	"github.com/cloudbase/garm/client/controller"
	"github.com/cloudbase/garm/client/controller_info"

	"github.com/mercedes-benz/garm-operator/pkg/metrics"
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

func (s *controllerClient) GetControllerInfo() (*controller_info.ControllerInfoOK, error) {
	return EnsureAuth(func() (*controller_info.ControllerInfoOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("controller.Info").Inc()
		controllerInfo, err := s.GarmAPI().ControllerInfo.ControllerInfo(&controller_info.ControllerInfoParams{}, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("controller.Info").Inc()
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
