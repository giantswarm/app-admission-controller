package app

import (
	"github.com/giantswarm/microerror"
)

var pspRemovalError = &microerror.Error{
	Kind: "pspRemovalError",
}

// IsPspRemoval asserts pspRemovalError.
func IsPspRemoval(err error) bool {
	return microerror.Cause(err) == pspRemovalError
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
