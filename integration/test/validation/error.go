package validation

import "github.com/giantswarm/microerror"

var executionFailedError = &microerror.Error{
	Kind: "executiomFailedError",
}