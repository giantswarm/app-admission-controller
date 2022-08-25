package app

import (
	"context"
	"github.com/giantswarm/app-admission-controller/internal/recorder"
	"github.com/giantswarm/app-admission-controller/pkg/project"
	"github.com/giantswarm/app/v6/pkg/validation"
	"github.com/giantswarm/k8sclient/v6/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	admissionv1 "k8s.io/api/admission/v1"
)

type InClusterAppValidatorConfig struct {
	Event recorder.Interface

	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	Provider string
}

type InClusterAppValidator struct {
	inClusterAppValidator *validation.Validator
	event                 recorder.Interface
	logger                micrologger.Logger
}

func NewInClusterAppValidator(config InClusterAppValidatorConfig) (*InClusterAppValidator, error) {
	if config.Event == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Event must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.Provider == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Provider must not be empty", config)
	}

	var err error

	var appValidator *validation.Validator
	{
		c := validation.Config{
			G8sClient: config.K8sClient.CtrlClient(),
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			ProjectName: project.Name(),
			Provider:    config.Provider,

			// `EnableManagedByLabel` enables skipping checks for
			// ConfigMap and Secret existence when the `giantswarm.io/managed-by`
			// label is present. This is used when app CRs are managed
			// with gitops tools like flux.
			EnableManagedByLabel: true,
		}
		appValidator, err = validation.NewValidator(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	v := &InClusterAppValidator{
		inClusterAppValidator: appValidator,
		event:                 config.Event,
		logger:                config.Logger,
	}

	return v, nil
}

func (v *InClusterAppValidator) Debugf(ctx context.Context, format string, params ...interface{}) {
	v.logger.WithIncreasedCallerDepth().Debugf(ctx, format, params...)
}

func (v *InClusterAppValidator) Errorf(ctx context.Context, err error, format string, params ...interface{}) {
	v.logger.WithIncreasedCallerDepth().Errorf(ctx, err, format, params...)
}

func (v *InClusterAppValidator) Resource() string {
	return Name
}

func (v *InClusterAppValidator) Validate(request *admissionv1.AdmissionRequest) (bool, error) {
	return false, nil
}
