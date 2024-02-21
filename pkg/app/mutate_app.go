package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-admission-controller/config"
	"github.com/giantswarm/app-admission-controller/pkg/mutator"
)

type MutatorConfig struct {
	K8sClient     k8sclient.Interface
	Logger        micrologger.Logger
	Provider      string
	ConfigPatches []config.ConfigPatch
}

type Mutator struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger
	// provider & configPatches are required by mutateConfigForPSPRemoval()
	provider      string
	configPatches []config.ConfigPatch
}

func NewMutator(config MutatorConfig) (*Mutator, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	mutator := &Mutator{
		k8sClient:     config.K8sClient,
		logger:        config.Logger,
		provider:      config.Provider,
		configPatches: config.ConfigPatches,
	}

	return mutator, nil
}

func (m *Mutator) Debugf(ctx context.Context, format string, params ...interface{}) {
	m.logger.WithIncreasedCallerDepth().Debugf(ctx, format, params...)
}

func (m *Mutator) Errorf(ctx context.Context, err error, format string, params ...interface{}) {
	m.logger.WithIncreasedCallerDepth().Errorf(ctx, err, format, params...)
}

func (m *Mutator) Mutate(request *admissionv1.AdmissionRequest) ([]mutator.PatchOperation, error) {
	ctx := context.Background()

	var result []mutator.PatchOperation

	if request.DryRun != nil && *request.DryRun {
		return result, nil
	}

	appNewCR := &v1alpha1.App{}
	if _, _, err := mutator.Deserializer.Decode(request.Object.Raw, nil, appNewCR); err != nil {
		return nil, microerror.Maskf(parsingFailedError, "unable to parse app: %#v", err)
	}

	appOldCR := &v1alpha1.App{}
	if _, _, err := mutator.Deserializer.Decode(request.OldObject.Raw, nil, appOldCR); err != nil {
		return nil, microerror.Maskf(parsingFailedError, "unable to parse app: %#v", err)
	}

	m.logger.Debugf(ctx, "mutating app %#q in namespace %#q", appNewCR.Name, appNewCR.Namespace)

	if request.Operation == admissionv1.Update && !appNewCR.DeletionTimestamp.IsZero() {
		m.logger.Debugf(ctx, "skipping mutation for UPDATE operation of app %#q in namespace %#q with non-zero deletion timestamp", appNewCR.Name, appNewCR.Namespace)
		return nil, nil
	}

	result, err := m.MutateApp(ctx, *appOldCR, *appNewCR, request.Operation)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	m.logger.Debugf(ctx, "applying %d patches to app %#q in namespace %#q", len(result), appNewCR.Name, appNewCR.Namespace)

	return result, nil
}

func (m *Mutator) MutateApp(ctx context.Context, oldApp, app v1alpha1.App, operation admissionv1.Operation) ([]mutator.PatchOperation, error) {
	var err error
	var result []mutator.PatchOperation

	isManagedInOrg := !key.InCluster(app) && key.IsInOrgNamespace(app)

	// Set empty labels and annotations in case they are not set. This is
	// in case we add new entries to null JSON objects. We don't want to do
	// this as needed because it can be potentially overwritten if set
	// after other patches.
	if len(app.Annotations) == 0 {
		result = append(result, mutator.PatchAdd("/metadata/annotations", map[string]string{}))
	}
	if len(app.Labels) == 0 {
		result = append(result, mutator.PatchAdd("/metadata/labels", map[string]string{}))
	}

	appVersionLabel := key.VersionLabel(app)
	if !isManagedInOrg && (appVersionLabel == "" || appVersionLabel == key.LegacyAppVersionLabel) {
		// We default to the same version as the chart-operator app CR
		// which means we don't need to check for a cluster CR.
		appVersionLabel, err = m.getChartOperatorAppVersion(ctx, app.Namespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		if appVersionLabel == "" {
			m.logger.Debugf(ctx, "skipping mutation of app %#q in namespace %#q due to missing version label", app.Name, app.Namespace)
			return nil, nil
		}
	}

	var patchLabels bool

	labelPatches, err := m.mutateLabels(ctx, app, appVersionLabel)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(labelPatches) > 0 {
		result = append(result, labelPatches...)
		patchLabels = true
	}

	ver, err := semver.NewVersion(appVersionLabel)
	if !isManagedInOrg && err != nil {
		m.logger.Debugf(ctx, "skipping mutation of app %#q in namespace %#q due to version label %#q", app.Name, app.Namespace, appVersionLabel)
		return nil, nil
	}

	// If the app CR does not have the unique version and is < 3.0.0 we skip
	// the defaulting logic apart from the labels. This is so the admission
	// controller is not enabled for existing platform releases.
	if !isManagedInOrg && key.VersionLabel(app) != uniqueAppCRVersion && ver.Major() < 3 {
		if patchLabels {
			m.logger.Debugf(ctx, "mutating only labels of app %#q in namespace %#q due to version label %#q", app.Name, app.Namespace, appVersionLabel)
			return result, nil
		}

		m.logger.Debugf(ctx, "skipping mutation of app %#q in namespace %#q due to version label %#q", app.Name, app.Namespace, appVersionLabel)
		return nil, nil
	}

	extraConfigPatches, err := m.mutateExtraConfigs(ctx, app)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Towards https://github.com/giantswarm/roadmap/issues/2716.
	// See method documentation for more details.
	pspConfigPatches, err := m.mutateConfigForPSPRemoval(ctx, app)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// This may seem as too much, but since both, the `extraConfigsPatches` and the
	// `pspConfigPatches`, are capable of adding patches to extra configs field,
	// it makes sense to me to extra the patch that initially prepares the field to the
	// parent method, hence it is here.
	// Note: I have thought about merging the two method together, but since the PSP
	// removal is the thing that already caused some tension, I would like to avoid
	// touching it altogether, because I believe what the method provides today has
	// been tested and works. It should not be difficult to merge the two, yet I
	// consciously choose to be paranoidical.
	needsExtraConfigsSet := len(key.ExtraConfigs(app)) == 0 && (hasPatchAddToExtraConfigs(extraConfigPatches) ||
		hasPatchAddToExtraConfigs(pspConfigPatches))

	if needsExtraConfigsSet {
		result = append(result, mutator.PatchAdd("/spec/extraConfigs", []v1alpha1.AppExtraConfig{}))
	}

	if len(extraConfigPatches) > 0 {
		result = append(result, extraConfigPatches...)
	}
	if len(pspConfigPatches) > 0 {
		result = append(result, pspConfigPatches...)
	}

	kubeConfigPatches, err := m.mutateKubeConfig(ctx, app)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(kubeConfigPatches) > 0 {
		result = append(result, kubeConfigPatches...)
	}

	return result, nil
}

func (m *Mutator) Resource() string {
	return Name
}

func (m *Mutator) getChartOperatorAppVersion(ctx context.Context, namespace string) (string, error) {
	var chartOperatorApp v1alpha1.App

	err := m.k8sClient.CtrlClient().Get(
		ctx,
		types.NamespacedName{Name: key.ChartOperatorAppName, Namespace: namespace},
		&chartOperatorApp)
	if apierrors.IsNotFound(err) {
		return "", nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	return key.VersionLabel(chartOperatorApp), nil
}

// The https://github.com/giantswarm/giantswarm/issues/29683 tightens the relations of App Platform
// and cluster values ConfigMaps as a provider of baseline configuration for apps. Due to this, the
// cluster ConfigMaps are to be always referenced in App CR for workload clusters. Because so far,
// the `.spec.config` field has not been truly reserved for this purpose, and hence could be used
// by user for other purposes, to avoid potential problems with making this reservation now, it has been
// decided to use the `.spec.extraConfigs` list and oblige the App Admission Controller to populate
// it with the cluster values.
func (m *Mutator) mutateExtraConfigs(ctx context.Context, app v1alpha1.App) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	// Return early if app is a Management Cluster app.
	if key.VersionLabel(app) == uniqueAppCRVersion {
		return nil, nil
	}

	clusterConfigMap := key.ClusterConfigMapName(app)

	// if submitted App CR is already configured with the cluster values ConfigMap
	// in the `.spec.config` or `.spec.extraConfigs` or `.spec.userConfig` fields,
	// we skip adding it to the `.spec.extraConfigs` list. If we in addition add
	// these values to the `.spec.extraConfigs` list it will only raise confusion,
	// see the linked issue.
	if key.AppConfigMapName(app) == clusterConfigMap && key.AppConfigMapNamespace(app) == app.Namespace {
		return nil, nil
	}

	for _, c := range key.ExtraConfigs(app) {
		if c.Name == clusterConfigMap && c.Namespace == app.Namespace {
			return nil, nil
		}
	}

	if key.UserConfigMapName(app) == clusterConfigMap && key.UserConfigMapNamespace(app) == app.Namespace {
		return nil, nil
	}

	result = append(result, mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
		Kind:      "configMap",
		Name:      clusterConfigMap,
		Namespace: app.Namespace,
		Priority:  bottomPriority,
	}))

	return result, nil
}

func (m *Mutator) mutateKubeConfig(ctx context.Context, app v1alpha1.App) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	// Return early if in-cluster is used.
	if key.InCluster(app) {
		return nil, nil
	}

	// Return early if either field is set.
	if key.KubeConfigSecretName(app) != "" || key.KubeConfigSecretNamespace(app) != "" {
		return nil, nil
	}

	kubeConfigNamespace, err := findKubeConfigNamespace(ctx, m.k8sClient.K8sClient(), app.Namespace, key.ClusterKubeConfigSecretName(app))
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if kubeConfigNamespace == "" {
		// Return early if we can't find a kubeconfig.
		return nil, nil
	}

	contextName := app.Namespace

	isManagedInOrg := !key.InCluster(app) && key.IsInOrgNamespace(app)
	if isManagedInOrg {
		contextName = key.ClusterLabel(app)
	}

	if key.KubeConfigContextName(app) == "" {
		result = append(result, mutator.PatchAdd("/spec/kubeConfig/context", map[string]string{
			"name": contextName,
		}))
	}

	result = append(result, mutator.PatchAdd("/spec/kubeConfig/secret", map[string]string{
		"namespace": kubeConfigNamespace,
		"name":      key.ClusterKubeConfigSecretName(app),
	}))

	return result, nil
}

func (m *Mutator) mutateLabels(ctx context.Context, app v1alpha1.App, appVersionLabel string) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	// Set app label if there is no app label present.
	if key.AppKubernetesNameLabel(app) == "" && key.AppLabel(app) == "" {
		result = append(result, mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), key.AppName(app)))
	}

	if (key.VersionLabel(app) == "" || key.VersionLabel(app) == key.LegacyAppVersionLabel) && appVersionLabel != "" {
		result = append(result, mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), appVersionLabel))
	}

	return result, nil
}

func findKubeConfigNamespace(ctx context.Context, k8sClient kubernetes.Interface, appNamespace, kubeConfigName string) (string, error) {
	_, err := k8sClient.CoreV1().Secrets(appNamespace).Get(ctx, kubeConfigName, metav1.GetOptions{})
	if err == nil {
		// kubeconfig is in the same namespace as the app CR.
		return appNamespace, nil
	}
	if apierrors.IsNotFound(err) {
		// If its not in the app CR namespace this may be a CAPI cluster.
		lo := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", "cluster.x-k8s.io/cluster-name", appNamespace),
		}
		secrets, err := k8sClient.CoreV1().Secrets(metav1.NamespaceAll).List(ctx, lo)
		if err != nil {
			return "", microerror.Mask(err)
		}

		for _, secret := range secrets.Items {
			if secret.Name == kubeConfigName {
				// We found it.
				return secret.Namespace, nil
			}
		}
	}

	// Empty return as we can't find a kubeconfig.
	return "", nil
}

func hasPatchAddToExtraConfigs(patches []mutator.PatchOperation) bool {
	for _, patch := range patches {
		if strings.HasPrefix(patch.Path, "/spec/extraConfigs/") {
			return true
		}
	}
	return false
}

func replaceToEscape(from string) string {
	return strings.Replace(from, "/", "~1", -1)
}
