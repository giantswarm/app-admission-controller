package version

import (
	"context"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type Version struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger
}

func New(config Config) (*Version, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	v := &Version{
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return v, nil
}

func (v *Version) GetReconcilingAppOperatorVersion(ctx context.Context, app v1alpha1.App) (*semver.Version, error) {
	raw, err := v.getReconcilingAppOperatorVersion(ctx, app)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if raw == "" {
		return nil, microerror.Maskf(notFoundError, "app-operator version label missing")
	}

	ver, err := semver.NewVersion(raw)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return ver, nil
}

func (v *Version) getReconcilingAppOperatorVersion(ctx context.Context, app v1alpha1.App) (string, error) {
	appOperatorVersion := key.VersionLabel(app)
	if appOperatorVersion != "" {
		return appOperatorVersion, nil
	}

	// If app-operator version label is not set in this CR fall back to the
	// value of the label of the chart-operator App CR in the same
	// namespace.
	chartOperatorApp, err := v.k8sClient.G8sClient().ApplicationV1alpha1().Apps(app.Namespace).Get(ctx, "chart-operator", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return "", nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	return key.VersionLabel(*chartOperatorApp), nil
}
