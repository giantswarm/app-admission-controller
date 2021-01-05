package app

import (
	"github.com/giantswarm/app/v4/pkg/validation"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type ValidatorConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type Validator struct {
	appValidator *validation.Validator
	logger       micrologger.Logger
}

func NewValidator(config ValidatorConfig) (*Validator, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var err error

	var appValidator *validation.Validator
	{
		c := validation.Config{
			G8sClient: config.K8sClient.G8sClient(),
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}
		appValidator, err = validation.NewValidator(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	validator := &Validator{
		appValidator: appValidator,
		logger:       config.Logger,
	}

	return validator, nil
}

func (v *Validator) Name() string {
	return Name
}
