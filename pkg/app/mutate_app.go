package app

import (
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/api/admission/v1beta1"

	"github.com/giantswarm/app-admission-controller/config"
	"github.com/giantswarm/app-admission-controller/pkg/mutator"
)

type Mutator struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger
}

func NewMutator(config config.Config) (*Mutator, error) {
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

func (m *Mutator) Mutate(request *v1beta1.AdmissionRequest) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	if request.DryRun != nil && *request.DryRun {
		return result, nil
	}

	/*
		appNewCR := &v1alpha1.App{}
		appOldCR := &v1alpha1.App{}
		if _, _, err := mutator.Deserializer.Decode(request.Object.Raw, nil, appNewCR); err != nil {
			return nil, microerror.Maskf(parsingFailedError, "unable to parse app: %#v", err)
		}
		if _, _, err := mutator.Deserializer.Decode(request.OldObject.Raw, nil, appOldCR); err != nil {
			return nil, microerror.Maskf(parsingFailedError, "unable to parse app: %#v", err)
		}

			patch := mutator.PatchReplace("/spec/replicas", replicas)
			result = append(result, patch)
	*/

	return result, nil
}

func (m *Mutator) Log(keyVals ...interface{}) {
	m.logger.Log(keyVals...)
}

func (m *Mutator) Resource() string {
	return Name
}
