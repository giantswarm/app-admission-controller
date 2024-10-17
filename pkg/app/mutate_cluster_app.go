package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	releases "github.com/giantswarm/release-operator/v3/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/app-admission-controller/pkg/mutator"
)

func (m *Mutator) mutateClusterApp(ctx context.Context, app v1alpha1.App) ([]mutator.PatchOperation, error) {
	// Check if app is a cluster-$provider app
	isClusterApp := (app.Spec.Catalog == "cluster" || app.Spec.Catalog == "cluster-test") && strings.HasPrefix(app.Spec.Name, "cluster-")
	if !isClusterApp {
		return nil, nil
	}

	m.logger.Debugf(ctx, "Cluster app mutation for setting App version based on the release. App/Cluster:%s, Namespace:%s\n",
		app.Name, app.Namespace)

	var clusterAppConfig map[string]interface{}

	// User values ConfigMap could be applied after the cluster-<provider> app manifest, so we retry 3 times here just
	// in case.
	getUserValues := func() error {
		var err error

		// First we have to take all possible cluster app configs and merge them into one. For that first we need the
		// cluster app catalog (so we can use existing app platform function to merge all configs).
		//
		// Here we have a little chicken & egg problem, because cluster app catalog value in App CR and in Release CR
		// can be different, so how to know which one to use before checking the Release CR, should we use cluster or
		// cluster-test catalog?
		// Answer - it doesn't really matter, because we're doing all of this here to get the release version, and
		// release version is cluster-specific, and it should never be set globally in the catalog values. So we just
		// use whatever catalog is set in the cluster App, and fallback to cluster catalog.
		catalogName := app.Spec.Catalog
		if catalogName == "" {
			catalogName = "cluster"
		}
		catalog := v1alpha1.Catalog{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "giantswarm",
				Name:      catalogName,
			},
		}
		err = m.k8sClient.CtrlClient().Get(ctx, client.ObjectKeyFromObject(&catalog), &catalog)
		if err != nil {
			return microerror.Mask(err)
		}
		clusterAppConfig, err = m.valuesService.MergeConfigMapData(ctx, app, catalog)
		if err != nil {
			return microerror.Mask(err)
		}
		return nil
	}
	b := backoff.NewMaxRetries(3, 5*time.Second)
	n := backoff.NewNotifier(m.logger, ctx)
	err := backoff.RetryNotify(getUserValues, b, n)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	globalValues, ok := clusterAppConfig["global"].(map[string]interface{})
	if !ok {
		return nil, microerror.Maskf(invalidConfigError, "global config not found in cluster app config")
	}

	releaseValuesObj, ok := globalValues["release"]
	if !ok {
		// If cluster app does not have release config set in user values, then we do nothing and just return.
		//
		// This way this cluster-<provider> app mutation does not affect existing clusters that do not use new releases.
		//
		// In case of new clusters that use new releases, release version Helm value is required in JSON schema so
		// cluster-<provider> Helm chart rendering will fail if release version is not set (which is expected and desired
		// behavior).
		return nil, nil
	}
	releaseValues, ok := releaseValuesObj.(map[string]interface{})
	if !ok {
		return nil, microerror.Maskf(invalidConfigError, "release config object is not a map")
	}
	releaseVersion, ok := releaseValues["version"].(string)
	if !ok {
		return nil, microerror.Maskf(invalidConfigError, "release version string is not found in release config")
	}

	// Now let's get the release resource from which we can read the cluster-$provider App version

	// remove "v" prefix from the release version, because Release CRs do not have it in the name
	releaseVersion = strings.TrimPrefix(releaseVersion, "v")

	// Provider name is based on the cluster app being used
	providerName := strings.ToLower(strings.TrimPrefix(app.Spec.Name, "cluster-"))
	releaseVersion = fmt.Sprintf("%s-%s", providerName, releaseVersion)

	// finally, get the Release resource
	var release releases.Release
	objectKey := client.ObjectKey{
		Name: releaseVersion,
	}
	err = m.k8sClient.CtrlClient().Get(ctx, objectKey, &release)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// and we get cluster-$provider app version
	var clusterAppCatalog string
	var clusterAppVersion string
	for _, component := range release.Spec.Components {
		if component.Name == app.Spec.Name {
			clusterAppCatalog = component.Catalog
			clusterAppVersion = component.Version
			break
		}
	}
	if clusterAppVersion == "" {
		return nil, microerror.Maskf(clusterAppVersionNotFound, "Cannot find the version of '%s' in the Release '%s/%s'", app.Spec.Name, app.Namespace, app.Name)
	}

	// Finally, create a patch for populating the cluster-$provider App version
	var result []mutator.PatchOperation
	if app.Spec.Version != clusterAppVersion {
		result = append(result, mutator.PatchAdd("/spec/version", clusterAppVersion))
	}
	if app.Spec.Catalog != clusterAppCatalog {
		result = append(result, mutator.PatchAdd("/spec/catalog", clusterAppCatalog))
	}

	return result, nil
}
