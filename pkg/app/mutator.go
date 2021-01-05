package app

import (
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type MutatorConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type Mutator struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger
}

func NewMutator(config MutatorConfig) (*Mutator, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	mutator := &Mutator{
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return mutator, nil
}

func (m *Mutator) Name() string {
	return Name
}
