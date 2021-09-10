package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned"
	"github.com/giantswarm/app/v5/pkg/key"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-admission-controller/pkg/mutator"
)

type MutatorConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type Mutator struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger
}

func NewMutator(config MutatorConfig) (*Mutator, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	mutator := &Mutator{
		k8sClient: config.K8sClient,
		logger:    config.Logger,
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
	if appOldCR == nil {
		appOldCR = &v1alpha1.App{}
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
	if appVersionLabel == "" || appVersionLabel == key.LegacyAppVersionLabel {
		// We default to the same version as the chart-operator app CR
		// which means we don't need to check for a cluster CR.
		appVersionLabel, err = getChartOperatorAppVersion(ctx, m.k8sClient.G8sClient(), app.Namespace)
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
	if err != nil {
		m.logger.Debugf(ctx, "skipping mutation of app %#q in namespace %#q due to version label %#q", app.Name, app.Namespace, appVersionLabel)
		return nil, nil
	}

	// If the app CR does not have the unique version and is < 3.0.0 we skip
	// the defaulting logic apart from the labels. This is so the admission
	// controller is not enabled for existing platform releases.
	if key.VersionLabel(app) != uniqueAppCRVersion && ver.Major() < 3 {
		if patchLabels {
			m.logger.Debugf(ctx, "mutating only labels of app %#q in namespace %#q due to version label %#q", app.Name, app.Namespace, appVersionLabel)
			return result, nil
		}

		m.logger.Debugf(ctx, "skipping mutation of app %#q in namespace %#q due to version label %#q", app.Name, app.Namespace, appVersionLabel)
		return nil, nil
	}

	configPatches, err := m.mutateConfig(ctx, app)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(configPatches) > 0 {
		result = append(result, configPatches...)
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

func (m *Mutator) mutateConfig(ctx context.Context, app v1alpha1.App) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	// Return early if either field is set.
	if key.AppConfigMapName(app) != "" || key.AppConfigMapNamespace(app) != "" {
		return nil, nil
	}

	// Return early if app is a Management Cluster app.
	if key.VersionLabel(app) == uniqueAppCRVersion {
		return nil, nil
	}

	// Return early if values configmap not found.
	_, err := m.k8sClient.K8sClient().CoreV1().ConfigMaps(app.Namespace).Get(ctx, key.ClusterConfigMapName(app), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}

	// If there is no secret then create a patch for the config block.
	if key.AppSecretName(app) == "" && key.AppSecretNamespace(app) == "" {
		result = append(result, mutator.PatchAdd("/spec/config", map[string]string{}))
	}

	result = append(result, mutator.PatchAdd("/spec/config/configMap", map[string]string{
		"namespace": app.Namespace,
		"name":      key.ClusterConfigMapName(app),
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

	if key.KubeConfigContextName(app) == "" {
		result = append(result, mutator.PatchAdd("/spec/kubeConfig/context", map[string]string{
			"name": app.Namespace,
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
	// Check for kubeconfig in the same namespace as the app CR.
	_, err := k8sClient.CoreV1().Secrets(appNamespace).Get(ctx, kubeConfigName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// If its not found this may be a CAPI cluster.
		lo := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", "cluster.x-k8s.io/cluster-name", appNamespace),
		}

		secrets, err := k8sClient.CoreV1().Secrets(metav1.NamespaceAll).List(ctx, lo)
		if err != nil {
			return "", microerror.Mask(err)
		}

		for _, secret := range secrets.Items {
			if secret.Name == kubeConfigName {
				return secret.Namespace, nil
			}
		}

		// Return empty as we can't find a kubeconfig.
		return "", nil
	}

	// kubeconfig is in the same namespace as the app CR.
	return appNamespace, nil
}

func getChartOperatorAppVersion(ctx context.Context, g8sClient versioned.Interface, namespace string) (string, error) {
	chartOperatorApp, err := g8sClient.ApplicationV1alpha1().Apps(namespace).Get(ctx, "chart-operator", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return "", nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	return key.VersionLabel(*chartOperatorApp), nil
}

func replaceToEscape(from string) string {
	return strings.Replace(from, "/", "~1", -1)
}
