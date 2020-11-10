package app

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
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

	result, err := m.MutateApp(ctx, *appNewCR, *appOldCR)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return result, nil
}

func (m *Mutator) MutateApp(ctx context.Context, appNewCR, appOldCR v1alpha1.App) ([]mutator.PatchOperation, error) {
	var result []mutator.PatchOperation

	configPatches, err := m.mutateConfig(ctx, appNewCR, appOldCR)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(configPatches) > 0 {
		result = append(result, configPatches...)
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

	clusterNamespace := appNewCR.Namespace
	clusterConfigMapName := fmt.Sprintf("%s-cluster-values", clusterNamespace)

	_, err := m.k8sClient.K8sClient().CoreV1().ConfigMaps(clusterNamespace).Get(ctx, clusterConfigMapName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// Fall through if configmap does not exist.
		return nil, nil
	}

	// If there is no secret then create a patch for the config block.
	if key.AppSecretName(appNewCR) == "" && key.AppSecretNamespace(appNewCR) == "" {
		result = append(result, mutator.PatchAdd("/spec/config", map[string]string{}))
	}

	result = append(result, mutator.PatchAdd("/spec/config/configMap", map[string]string{}))
	result = append(result, mutator.PatchAdd("/spec/config/configMap/namespace", clusterNamespace))
	result = append(result, mutator.PatchAdd("/spec/config/configMap/name", clusterConfigMapName))

	return result, nil
}
