package app

import (
	"context"
	"strings"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	releases "github.com/giantswarm/release-operator/v3/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/app-admission-controller/pkg/mutator"
)

func (m *Mutator) mutateClusterApp(ctx context.Context, app v1alpha1.App) ([]mutator.PatchOperation, error) {
	m.logger.Debugf(ctx, "Cluster app mutation for setting App version based on the release. App/Cluster:%s, Namespace:%s\n",
		app.Name, app.Namespace)

	// Check if app is a cluster-$provider app
	isClusterApp := app.Spec.Catalog == "cluster" && strings.HasPrefix(app.Spec.Name, "cluster-")
	if !isClusterApp {
		return nil, nil
	}

	// Do nothing if the App version is already set
	if app.Spec.Version != "" {
		return nil, nil
	}

	// Now let's get the release resource from which we can read the cluster-$provider App version
	releaseVersion, ok := app.Labels[label.ReleaseVersion]
	if !ok {
		return nil, microerror.Maskf(releaseVersionNotSpecified, "Release version label '%s' not set in cluster App CR '%s/%s'", label.ReleaseVersion, app.Namespace, app.Name)
	}

	var release releases.Release
	objectKey := client.ObjectKey{
		Name: releaseVersion,
	}
	err := m.k8sClient.CtrlClient().Get(ctx, objectKey, &release)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// and we get cluster-$provider app version
	var clusterAppVersion string
	for _, component := range release.Spec.Components {
		if component.Name == app.Spec.Name {
			clusterAppVersion = component.Version
			break
		}
	}
	if clusterAppVersion == "" {
		return nil, microerror.Maskf(clusterAppVersionNotFound, "Cannot find the version of '%s' in the Release '%s/%s'", app.Spec.Name, app.Namespace, app.Name)
	}

	// Finally, create a patch for populating the cluster-$provider App version
	result := []mutator.PatchOperation{
		mutator.PatchAdd("/spec/version", clusterAppVersion),
	}

	return result, nil
}
