package inspector

import (
	"github.com/giantswarm/microerror"
)

var securityViolationError = &microerror.Error{
	Kind: "securityViolationError",
}

// IsValidationError asserts validationError.
func IsSecurityViolationError(err error) bool {
	return microerror.Cause(err) == securityViolationError
}
