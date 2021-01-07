package app

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v4/pkg/validation"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/api/admission/v1beta1"

	"github.com/giantswarm/app-admission-controller/pkg/app/internal/version"
	"github.com/giantswarm/app-admission-controller/pkg/validator"
)

const (
	Name = "app"

	uniqueOperatorVersion = "0.0.0"
)

type ValidatorConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type Validator struct {
	appValidator *validation.Validator
	logger       micrologger.Logger
	version      version.Interface
}

func NewValidator(config ValidatorConfig) (*Validator, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var err error

	var newVersion version.Interface
	{
		c := version.Config(config)
		newVersion, err = version.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

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
		version:      newVersion,
	}

	return validator, nil
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

func (v *Validator) Validate(request *v1beta1.AdmissionRequest) (bool, error) {
	var err error

	ctx := context.Background()

	var app v1alpha1.App

	if _, _, err := validator.Deserializer.Decode(request.Object.Raw, nil, &app); err != nil {
		return false, microerror.Maskf(parsingFailedError, "unable to parse app: %#v", err)
	}

	v.logger.Debugf(ctx, "validating app %#q in namespace %#q", app.Name, app.Namespace)

	// We check the deletion timestamp because app CRs may be deleted by
	// deleting the namespace they belong to.
	if !app.DeletionTimestamp.IsZero() {
		v.logger.Debugf(ctx, "admitted deletion of app %#q in namespace %#q", app.Name, app.Namespace)
		return true, nil
	}

	appOperatorVersion, err := v.version.GetReconcilingAppOperatorVersion(ctx, app)
	if version.IsNotFound(err) {
		v.logger.Debugf(ctx, "skipping validation of app %#q in namespace %#q due to missing app-operator version label", app.Name, app.Namespace)
		return true, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	// If the app CR does not have the unique version and is < 3.0.0 we skip
	// the validation logic. This is so the admission controller is not
	// enabled for existing platform releases.
	if appOperatorVersion.Original() != uniqueOperatorVersion && appOperatorVersion.Major() < 3 {
		v.logger.Debugf(ctx, "skipping validation of app %#q in namespace %#q due to app-operator version label %#q", app.Name, app.Namespace, appOperatorVersion.Original())
		return true, nil
	}

	appAllowed, err := v.appValidator.ValidateApp(ctx, app)
	if validation.IsAppDependencyNotReady(err) {
		v.logger.Debugf(ctx, "skipping validation of app %#q in namespace %#q due to app dependency not ready yet", app.Name, app.Namespace)
		return true, nil
	} else if err != nil {
		v.logger.Errorf(ctx, err, "rejected app %#q in namespace %#q", app.Name, app.Namespace)
		return false, microerror.Mask(err)
	}

	v.logger.Debugf(ctx, "admitted app %#q in namespace %#q", app.Name, app.Namespace)

	return appAllowed, nil
}
