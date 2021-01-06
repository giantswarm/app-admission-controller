package app

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app-admission-controller/pkg/admissionkit/pkg/validator"
	"github.com/giantswarm/microerror"
)

func (v *Validator) Validate(ctx context.Context, req validator.Request) (*validator.Response, error) {
	app, err := toAppCR(req.Obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	validation, err := v.validate(ctx, app)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	resp := &validator.Response{
		Validation: validation,
	}

	return resp, nil
}

func (v *Validator) validate(ctx context.Context, app *v1alpha1.App) (*validator.Validation, error) {
	// https://github.com/giantswarm/app-admission-controller/blob/master/pkg/app/validate_app.go#L92
	return validator.Accept(), nil
}
