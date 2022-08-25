package app

import (
	"context"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app-admission-controller/internal/recorder"
	"github.com/giantswarm/app-admission-controller/pkg/project"
	"github.com/giantswarm/app-admission-controller/pkg/validator"
	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/app/v6/pkg/validation"
	"github.com/giantswarm/k8sclient/v6/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type InClusterAppValidatorConfig struct {
	Event recorder.Interface

	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	Provider string
}

type InClusterAppValidator struct {
	inClusterAppValidator *validation.Validator
	client                k8sclient.Interface
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
		client:                config.K8sClient,
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
	ctx := context.Background()

	var app v1alpha1.App

	// Deserialize the object into an app
	if _, _, err := validator.Deserializer.Decode(request.Object.Raw, nil, &app); err != nil {
		return false, microerror.Maskf(parsingFailedError, "unable to parse app: %#v", err)
	}

	// We only validate in-cluster apps here, so bail out early if it is not one of them
	if !key.InCluster(app) {
		v.logger.Debugf(ctx, "skipping validation of app %#q in namespace %#q due to not being an in-cluster app", app.Name, app.Namespace, key.VersionLabel(app))
		return true, nil
	}

	// Skip validation if the app was updated with a non-zero deletion timestamp
	if request.Operation == admissionv1.Update && !app.DeletionTimestamp.IsZero() {
		v.logger.Debugf(ctx, "skipping validation for UPDATE operation of app %#q in namespace %#q with non-zero deletion timestamp", app.Name, app.Namespace)
		return true, nil
	}

	// Let's log users names and groups membership in case we need to
	// troubleshoot possible problems. This way it will be much easier
	// to recognize the actor.
	v.logger.Debugf(
		ctx,
		"validating action taken by `%s` user in `%s` groups",
		request.UserInfo.Username,
		strings.Join(request.UserInfo.Groups, ","),
	)

	// Let's do the actual validation
	apps := &v1alpha1.AppList{}

	// TODO Explain algorithm
	fieldName := "metadata.name"
	fieldValue := app.Name
	err := v.client.CtrlClient().List(ctx, apps, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(fieldName, fieldValue),
	})
	if err != nil {
		return false, microerror.Maskf(listingAppsFailed, "failed to list apps with %#q set to %#q, %#v", fieldName, fieldValue, err)
	}

	for _, inspectedApp := range apps.Items {
		// If it is the same app (we are handling an update event for example) then skip over
		if inspectedApp.Namespace == app.Namespace {
			continue
		}

		// Found another app that is in-cluster and bears the same name, you shall not pass!
		return false, microerror.Mask(validationError)
	}

	// All is good, let it pass
	return true, nil
}
