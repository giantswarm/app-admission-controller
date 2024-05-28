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
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/app-admission-controller/pkg/mutator"
)

func (m *Mutator) mutateClusterApp(ctx context.Context, app v1alpha1.App) ([]mutator.PatchOperation, error) {
	m.logger.Debugf(ctx, "Cluster app mutation for setting App version based on the release. App/Cluster:%s, Namespace:%s\n",
		app.Name, app.Namespace)

	// Check if app is a cluster-$provider app
	isClusterApp := (app.Spec.Catalog == "cluster" || app.Spec.Catalog == "cluster-test") && strings.HasPrefix(app.Spec.Name, "cluster-")
	if !isClusterApp {
		return nil, nil
	}

	// Do nothing if the App version is already set
	// TODO: fix this and allow for version updates
	if app.Spec.Version != "" {
		return nil, nil
	}

	userConfigMapName := app.Spec.UserConfig.ConfigMap.Name
	if userConfigMapName == "" {
		return nil, microerror.Maskf(clusterAppUserConfigNotSet, "Cluster App '%s/%s does not have the user config", app.Namespace, app.Name)
	}
	userConfigMapNameSpace := app.Spec.UserConfig.ConfigMap.Namespace
	if userConfigMapNameSpace == "" {
		userConfigMapNameSpace = app.Namespace
	}

	// Now let's get the release resource from which we can read the cluster-$provider App version

	// First we need user config, to get release version from it.
	var userValuesConfigMap *v1.ConfigMap
	getUserValues := func() error {
		var err error
		userValuesConfigMap, err = m.k8sClient.K8sClient().CoreV1().ConfigMaps(userConfigMapNameSpace).Get(ctx, userConfigMapName, metav1.GetOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		return nil
	}
	b := backoff.NewConstant(15*time.Second, 5*time.Second)
	n := backoff.NewNotifier(m.logger, ctx)
	err := backoff.RetryNotify(getUserValues, b, n)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var userValues struct {
		Global struct {
			Release struct {
				Version string `yaml:"version"`
			} `yaml:"release"`
		} `yaml:"global"`
	}
	err = yaml.Unmarshal([]byte(userValuesConfigMap.Data["values"]), &userValues)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if userValues.Global.Release.Version == "" {
		return nil, microerror.Maskf(releaseVersionNotSpecified, "Release version not set in cluster App '%s/%s' user value global.release.version", app.Namespace, app.Name)
	}

	// ensure that release version has "v" prefix, because Release CRs have it in the name
	releaseVersion := strings.TrimPrefix(userValues.Global.Release.Version, "v")
	releaseVersion = fmt.Sprintf("v%s", releaseVersion)

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
