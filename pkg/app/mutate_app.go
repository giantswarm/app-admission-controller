package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/app/v3/pkg/key"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/api/admission/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

func (m *Mutator) Log(keyVals ...interface{}) {
	m.logger.Log(keyVals...)
}

func (m *Mutator) Mutate(request *v1beta1.AdmissionRequest) ([]mutator.PatchOperation, error) {
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

	m.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("mutating app %#q in namespace %#q", appNewCR.Name, appNewCR.Namespace))

	// We check the deletion timestamp because app CRs may be deleted by
	// deleting the namespace they belong to.
	if !appNewCR.DeletionTimestamp.IsZero() {
		m.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("admitted deletion of app %#q in namespace %#q", appNewCR.Name, appNewCR.Namespace))
		return nil, nil
	}

	result, err := m.MutateApp(ctx, *appNewCR, *appOldCR)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	m.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("applying %d patches to app %#q in namespace %#q", len(result), appNewCR.Name, appNewCR.Namespace))

	return result, nil
}

func (m *Mutator) MutateApp(ctx context.Context, appNewCR, appOldCR v1alpha1.App) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	labelPatches, err := m.mutateLabels(ctx, appNewCR, appOldCR)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(labelPatches) > 0 {
		result = append(result, labelPatches...)
	}

	configPatches, err := m.mutateConfig(ctx, appNewCR, appOldCR)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(configPatches) > 0 {
		result = append(result, configPatches...)
	}

	kubeConfigPatches, err := m.mutateKubeConfig(ctx, appNewCR, appOldCR)
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

func (m *Mutator) mutateConfig(ctx context.Context, appNewCR, appOldCR v1alpha1.App) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	// Return early if either field is set.
	if key.AppConfigMapName(appNewCR) != "" || key.AppConfigMapNamespace(appNewCR) != "" {
		return nil, nil
	}

	// If there is no secret then create a patch for the config block.
	if key.AppSecretName(appNewCR) == "" && key.AppSecretNamespace(appNewCR) == "" {
		result = append(result, mutator.PatchAdd("/spec/config", map[string]string{}))
	}

	result = append(result, mutator.PatchAdd("/spec/config/configMap", map[string]string{}))
	result = append(result, mutator.PatchAdd("/spec/config/configMap/namespace", appNewCR.Namespace))
	result = append(result, mutator.PatchAdd("/spec/config/configMap/name", key.ClusterConfigMapName(appNewCR)))

	return result, nil
}

func (m *Mutator) mutateKubeConfig(ctx context.Context, appNewCR, appOldCR v1alpha1.App) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	// Return early if in-cluster is used.
	if key.InCluster(appNewCR) {
		return nil, nil
	}

	// Return early if either field is set.
	if key.KubeConfigSecretName(appNewCR) != "" || key.KubeConfigSecretNamespace(appNewCR) != "" {
		return nil, nil
	}

	if key.KubeConfigContextName(appNewCR) == "" {
		result = append(result, mutator.PatchAdd("/spec/kubeConfig/context", map[string]string{}))
		result = append(result, mutator.PatchAdd("/spec/kubeConfig/context/name", appNewCR.Namespace))
	}

	result = append(result, mutator.PatchAdd("/spec/kubeConfig/secret", map[string]string{}))
	result = append(result, mutator.PatchAdd("/spec/kubeConfig/secret/namespace", appNewCR.Namespace))
	result = append(result, mutator.PatchAdd("/spec/kubeConfig/secret/name", key.ClusterKubeConfigSecretName(appNewCR)))

	return result, nil
}

func (m *Mutator) mutateLabels(ctx context.Context, appNewCR, appOldCR v1alpha1.App) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	// Set app label if there is no app label present.
	if key.AppKubernetesNameLabel(appNewCR) == "" && key.AppLabel(appNewCR) == "" {
		result = append(result, mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", label.AppKubernetesName), key.AppName(appNewCR)))
	}

	// Set version label to be the same as the chart-operator app CR. This
	// is the version we need and means we don't need to check for a cluster CR.
	if key.VersionLabel(appNewCR) == "" {
		appVersion, err := getChartOperatorAppVersion(ctx, m.k8sClient.G8sClient(), appNewCR.Namespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		if appVersion != "" {
			result = append(result, mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), appVersion))
		}
	}

	if len(appNewCR.Labels) == 0 {
		root := mutator.PatchAdd("/metadata/labels", map[string]string{})
		result = append([]mutator.PatchOperation{root}, result...)
	}

	return result, nil
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
