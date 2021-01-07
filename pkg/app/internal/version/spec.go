package version

import (
	"context"

	"github.com/Masterminds/semver/v3"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
)

type Interface interface {
	// GetReconcilingAppOperatorVersion retrieves version of the
	// app-operator reconciling this App CR. It tries to get the version
	// from the app-operator.giantswarm.io/version label of the given App
	// CR. If the label is not present if falls back to the same label in
	// chart-operator App CR from the same namespace as the given App CR.
	//
	// It returns error matched by IsNotFound if the label is not found.
	GetReconcilingAppOperatorVersion(ctx context.Context, app v1alpha1.App) (*semver.Version, error)
}
