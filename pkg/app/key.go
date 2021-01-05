package app

import (
	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
)

const (
	// Name of the mutator and validator in this package.
	Name = "app"
)

func toAppCR(v interface{}) (*v1alpha1.App, error) {
	if v == nil {
		return nil, microerror.Maskf(wrongTypeError, "expected non-nil, got %#v", v)
	}

	p, ok := v.(*v1alpha1.App)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected %T, got %T", p, v)
	}

	deepCopy := p.DeepCopy()

	return deepCopy, nil
}
