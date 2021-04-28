package app

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/app/v4/pkg/validation"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	admissionv1 "k8s.io/api/admission/v1"

	"github.com/giantswarm/app-admission-controller/internal/recorder"
	"github.com/giantswarm/app-admission-controller/pkg/validator"
)

const (
	Name = "app"

	uniqueAppCRVersion = "0.0.0"
)

type ValidatorConfig struct {
	Event     recorder.Interface
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	Provider string
}

type Validator struct {
	appValidator *validation.Validator
	event        recorder.Interface
	logger       micrologger.Logger
}

func NewValidator(config ValidatorConfig) (*Validator, error) {
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
			G8sClient: config.K8sClient.G8sClient(),
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			Provider: config.Provider,
		}
		appValidator, err = validation.NewValidator(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	v := &Validator{
		appValidator: appValidator,
		event:        config.Event,
		logger:       config.Logger,
	}

	return v, nil
}

func (v *Validator) Debugf(ctx context.Context, format string, params ...interface{}) {
	v.logger.WithIncreasedCallerDepth().Debugf(ctx, format, params...)
}

func (v *Validator) Errorf(ctx context.Context, err error, format string, params ...interface{}) {
	v.logger.WithIncreasedCallerDepth().Errorf(ctx, err, format, params...)
}

func (v *Validator) Resource() string {
	return Name
}

func (v *Validator) Validate(request *admissionv1.AdmissionRequest) (bool, error) {
	ctx := context.Background()

	var app v1alpha1.App

	if _, _, err := validator.Deserializer.Decode(request.Object.Raw, nil, &app); err != nil {
		return false, microerror.Maskf(parsingFailedError, "unable to parse app: %#v", err)
	}

	v.logger.Debugf(ctx, "validating app %#q in namespace %#q", app.Name, app.Namespace)

	if request.Operation == admissionv1.Update && !app.DeletionTimestamp.IsZero() {
		v.logger.Debugf(ctx, "skipping validation for UPDATE operation of app %#q in namespace %#q with non-zero deletion timestamp", app.Name, app.Namespace)
		return true, nil
	}

	ver, err := semver.NewVersion(key.VersionLabel(app))
	if err != nil {
		v.logger.Debugf(ctx, "skipping validation of app %#q in namespace %#q due to version label %#q", app.Name, app.Namespace, key.VersionLabel(app))
		return true, nil
	}

	// If the app CR does not have the unique version and is < 3.0.0 we skip
	// the validation logic. This is so the admission controller is not
	// enabled for existing platform releases.
	if key.VersionLabel(app) != uniqueAppCRVersion && ver.Major() < 3 {
		v.logger.Debugf(ctx, "skipping validation of app %#q in namespace %#q due to version label %#q", app.Name, app.Namespace, key.VersionLabel(app))
		return true, nil
	}

	appAllowed, err := v.appValidator.ValidateApp(ctx, app)
	if err != nil {
		v.logger.Errorf(ctx, err, "rejected app %#q in namespace %#q", app.Name, app.Namespace)
		return false, microerror.Mask(err)
	}

	err = v.emitEvents(ctx, request, app)
	if err != nil {
		v.logger.Errorf(ctx, err, "app %#q has failed to emit events", app.Name)
	}

	v.logger.Debugf(ctx, "admitted app %#q in namespace %#q", app.Name, app.Namespace)

	return appAllowed, nil
}

func (v *Validator) emitEvents(ctx context.Context, request *admissionv1.AdmissionRequest, app v1alpha1.App) error {
	if request.Operation != admissionv1.Update {
		// no-op when it's not update
		return nil
	}

	var oldApp v1alpha1.App

	if _, _, err := validator.Deserializer.Decode(request.OldObject.Raw, nil, &oldApp); err != nil {
		return microerror.Maskf(parsingFailedError, "unable to parse app: %#v", err)
	}

	compareFunc := map[string]func(v1alpha1.App) string{
		"version":    key.Version,
		"appCatalog": key.CatalogName,
		"userConfigMap": func(app v1alpha1.App) string {
			return fmt.Sprintf("%s/%s", key.UserConfigMapNamespace(app), key.UserConfigMapName(app))
		},
		"userSecret": func(app v1alpha1.App) string {
			return fmt.Sprintf("%s/%s", key.UserSecretNamespace(app), key.UserSecretName(app))
		},
		"appConfigMap": func(app v1alpha1.App) string {
			return fmt.Sprintf("%s/%s", key.AppConfigMapNamespace(app), key.AppConfigMapName(app))
		},
		"appSecret": func(app v1alpha1.App) string {
			return fmt.Sprintf("%s/%s", key.AppSecretNamespace(app), key.AppSecretName(app))
		},
	}

	for name, f := range compareFunc {
		if f(app) != f(oldApp) {
			v.event.Emit(ctx, &app, "AppUpdated", "%s has been changed to %#q", name, f(app))
		}
	}

	return nil
}
