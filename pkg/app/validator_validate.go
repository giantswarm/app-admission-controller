package app

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app-admission-controller/pkg/admissionkit/pkg/validator"
	"github.com/giantswarm/app-admission-controller/pkg/app/internal/version"
	"github.com/giantswarm/app/v3/pkg/validation"
	"github.com/giantswarm/microerror"
)

func (v *Validator) Validate(ctx context.Context, req validator.Request) (*validator.Response, error) {
	app, err := toAppCR(req.Obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	rejection, err := v.validate(ctx, *app)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	resp := &validator.Response{
		Rejection: rejection,
	}

	return resp, nil
}

func (v *Validator) validate(ctx context.Context, app v1alpha1.App) (*validator.Rejection, error) {
	// We check the deletion timestamp because app CRs may be deleted by
	// deleting the namespace they belong to.
	if !app.DeletionTimestamp.IsZero() {
		v.logger.Debugf(ctx, "admitted deletion of app %#q in namespace %#q", app.Name, app.Namespace)
		return nil, nil
	}

	appOperatorVersion, err := v.version.GetReconcilingAppOperatorVersion(ctx, app)
	if version.IsNotFound(err) {
		v.logger.Debugf(ctx, "cancelling validator due to missing app-operator version label")
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	// If the app CR does not have the unique version and is < 3.0.0 we skip
	// the validation logic. This is so the admission controller is not
	// enabled for existing platform releases.
	if appOperatorVersion.Major() < 3 {
		v.logger.Debugf(ctx, "cancelling validator due to app-operator version = %#q lower than 3.0.0", appOperatorVersion.Original())
		return nil, nil
	}

	_, err = v.appValidator.ValidateApp(ctx, app)
	if validation.IsAppConfigMapNotFound(err) {
		// Fall trough.
	} else if validation.IsKubeConfigNotFound(err) {
		// Fall trough.
	} else if validation.IsValidationError(err) {
		return validator.Rejectf("validation error: %s", err), nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return nil, nil
}
