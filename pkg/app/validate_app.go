package app

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v3/validation"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/api/admission/v1beta1"

	"github.com/giantswarm/app-admission-controller/pkg/validator"
)

const (
	Name = "app"
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

func (v *Validator) Log(keyVals ...interface{}) {
	v.logger.Log(keyVals...)
}

func (v *Validator) Resource() string {
	return Name
}

func (v *Validator) Validate(request *v1beta1.AdmissionRequest) (bool, error) {
	ctx := context.Background()

	var app v1alpha1.App

	if _, _, err := validator.Deserializer.Decode(request.Object.Raw, nil, &app); err != nil {
		return false, microerror.Maskf(parsingFailedError, "unable to parse app: %#v", err)
	}

	v.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("validating app %#q in namespace %#q", app.Name, app.Namespace))

	// We check the deletion timestamp rather than for a delete event because
	// app CRs may be deleted by deleting the namespace they belong to.
	if !app.DeletionTimestamp.IsZero() {
		v.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("admitted deleted app %#q in namespace %#q", app.Name, app.Namespace))
		return true, nil
	}

	appAllowed, err := v.appValidator.ValidateApp(ctx, app)
	if err != nil {
		v.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("rejected app %#q in namespace %#q", app.Name, app.Namespace), "stack", microerror.JSON(err))
		return false, microerror.Mask(err)
	}

	v.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("admitted app %#q in namespace %#q", app.Name, app.Namespace))

	return appAllowed, nil
}
