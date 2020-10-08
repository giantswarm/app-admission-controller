package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v2/pkg/label"
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/api/admission/v1beta1"

	"github.com/giantswarm/app-admission-controller/config"
	"github.com/giantswarm/app-admission-controller/pkg/key"
	"github.com/giantswarm/app-admission-controller/pkg/mutator"
)

type Mutator struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger
}

func NewMutator(config config.Config) (*Mutator, error) {
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

	result, err := m.MutateApp(ctx, *appNewCR, *appOldCR)
	if err != nil {
		return nil, microerror.Mask(err)
	}

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

	if key.AppConfigMapName(appNewCR) == "" && key.AppConfigMapNamespace(appNewCR) == "" {
		result = append(result, mutator.PatchAdd("/spec/config", map[string]string{}))
		result = append(result, mutator.PatchAdd("/spec/config/configMap", map[string]string{}))
		result = append(result, mutator.PatchAdd("/spec/config/configMap/namespace", appNewCR.Namespace))
		result = append(result, mutator.PatchAdd("/spec/config/configMap/name", fmt.Sprintf("%s-cluster-values", appNewCR.Namespace)))
	}

	return result, nil
}

func (m *Mutator) mutateKubeConfig(ctx context.Context, appNewCR, appOldCR v1alpha1.App) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	if !key.InCluster(appNewCR) {
		result = append(result, mutator.PatchAdd("/spec/kubeConfig", map[string]string{}))

		if key.KubeConfigContextName(appNewCR) == "" {
			result = append(result, mutator.PatchAdd("/spec/kubeConfig/context", map[string]string{}))
			result = append(result, mutator.PatchAdd("/spec/kubeConfig/context/name", appNewCR.Namespace))
		}

		if key.KubeConfigSecretName(appNewCR) == "" && key.KubeConfigSecretNamespace(appNewCR) == "" {
			result = append(result, mutator.PatchAdd("/spec/kubeConfig/secret", map[string]string{}))
			result = append(result, mutator.PatchAdd("/spec/kubeConfig/secret/name", fmt.Sprintf("%s-kubeconfig", appNewCR.Namespace)))
			result = append(result, mutator.PatchAdd("/spec/kubeConfig/secret/namespace", appNewCR.Namespace))
		}
	}

	if key.AppConfigMapName(appNewCR) == "" && key.AppConfigMapNamespace(appNewCR) == "" {
		result = append(result, mutator.PatchAdd("/spec/config", map[string]string{}))
		result = append(result, mutator.PatchAdd("/spec/config/configMap", map[string]string{}))
		result = append(result, mutator.PatchAdd("/spec/config/configMap/namespace", appNewCR.Namespace))
		result = append(result, mutator.PatchAdd("/spec/config/configMap/name", fmt.Sprintf("%s-cluster-values", appNewCR.Namespace)))
	}

	return result, nil
}

func (m *Mutator) mutateLabels(ctx context.Context, appNewCR, appOldCR v1alpha1.App) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	if key.VersionLabel(appNewCR) == "" {
		version := "1.0.0"

		if appNewCR.Namespace == "giantswarm" {
			version = "0.0.0"
		}

		if len(appNewCR.Labels) == 0 {
			result = append(result, mutator.PatchAdd("/metadata/labels", map[string]string{}))
		}

		patch := mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), version)
		result = append(result, patch)
	}

	return result, nil
}

func replaceToEscape(from string) string {
	return strings.Replace(from, "/", "~1", -1)
}
