package inspector

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var securityViolationError = &microerror.Error{
	Kind: "securityViolationError",
}

// IsSecurityViolationError asserts validationError.
func IsSecurityViolationError(err error) bool {
	return microerror.Cause(err) == securityViolationError
}
