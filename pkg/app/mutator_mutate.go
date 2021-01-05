package app

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app-admission-controller/pkg/admissionkit/pkg/mutator"
	"github.com/giantswarm/microerror"
)

func (m *Mutator) Mutate(ctx context.Context, req mutator.Request) (*mutator.Response, error) {
	app, err := toAppCR(req.Obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	err = m.mutate(ctx, app)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	resp := &mutator.Response{
		MutatedObj: app,
	}

	return resp, nil
}

func (m *Mutator) mutate(ctx context.Context, app *v1alpha1.App) error {
	if len(app.Labels) == 0 {
		app.Labels = map[string]string{}
	}

	app.Labels["hackathon"] = "Q4"

	// return nil
	return microerror.Maskf(invalidConfigError, "TEST ERROR PAWEL")
}
