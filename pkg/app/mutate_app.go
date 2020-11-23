package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
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

	result, err := m.MutateApp(ctx, *appNewCR)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	m.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("applying %d patches to app %#q in namespace %#q", len(result), appNewCR.Name, appNewCR.Namespace))

	return result, nil
}

func (m *Mutator) MutateApp(ctx context.Context, app v1alpha1.App) ([]mutator.PatchOperation, error) {
	var err error
	var result []mutator.PatchOperation

	appVersionLabel := key.VersionLabel(app)
	if appVersionLabel == "" {
		// If there is no version label check the value for the chart-operator
		// app CR. This is the version we need and means we don't need to check
		// for a cluster CR.
		appVersionLabel, err = getChartOperatorAppVersion(ctx, m.k8sClient.G8sClient(), app.Namespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		if appVersionLabel == "" {
			m.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("skipping mutation of app %#q in namespace %#q due to missing version label", app.Name, app.Namespace))
			return nil, nil
		}
	}

	ver, err := semver.NewVersion(appVersionLabel)
	if err != nil {
		m.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("skipping mutation of app %#q in namespace %#q due to version label %#q", app.Name, app.Namespace, appVersionLabel))
		return nil, nil
	}

	if ver.Major() < 3 {
		m.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("skipping mutation of app %#q in namespace %#q due to version label %#q", app.Name, app.Namespace, appVersionLabel))
		return nil, nil
	}

	labelPatches, err := m.mutateLabels(ctx, app, appVersionLabel)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(labelPatches) > 0 {
		result = append(result, labelPatches...)
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

	if key.KubeConfigContextName(app) == "" {
		result = append(result, mutator.PatchAdd("/spec/kubeConfig/context", map[string]string{
			"name": app.Namespace,
		}))
	}

	result = append(result, mutator.PatchAdd("/spec/kubeConfig/secret", map[string]string{
		"namespace": app.Namespace,
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

	if key.VersionLabel(app) == "" && appVersionLabel != "" {
		result = append(result, mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), appVersionLabel))
	}

	if len(app.Labels) == 0 {
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
