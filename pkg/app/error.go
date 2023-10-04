package app

import (
	"github.com/giantswarm/microerror"
)

var pssComplianceError = &microerror.Error{
	Kind: "pssComplianceError",
}

// IsPssCompliance asserts pssComplianceError.
func IsPssCompliance(err error) bool {
	return microerror.Cause(err) == pssComplianceError
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var parsingFailedError = &microerror.Error{
	Kind: "parsingFailedError",
}

// IsParsingFailed asserts parsingFailedError.
func IsParsingFailed(err error) bool {
	return microerror.Cause(err) == parsingFailedError
}
