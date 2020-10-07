package key

import (
	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v2/pkg/label"
)

// AppConfigMapName returns the name of the configmap that stores app level
// config for the provided app CR.
func AppConfigMapName(customResource v1alpha1.App) string {
	return customResource.Spec.Config.ConfigMap.Name
}

// AppConfigMapNamespace returns the namespace of the configmap that stores app
// level config for the provided app CR.
func AppConfigMapNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.Config.ConfigMap.Namespace
}

// AppSecretName returns the name of the secret that stores app level
// secrets for the provided app CR.
func AppSecretName(customResource v1alpha1.App) string {
	return customResource.Spec.Config.Secret.Name
}

// AppSecretNamespace returns the namespace of the secret that stores app
// level secrets for the provided app CR.
func AppSecretNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.Config.Secret.Namespace
}

func ClusterID(customResource v1alpha1.App) string {
	return customResource.GetLabels()[label.Cluster]
}

func InCluster(customResource v1alpha1.App) bool {
	return customResource.Spec.KubeConfig.InCluster
}

func IsDeleted(customResource v1alpha1.App) bool {
	return customResource.DeletionTimestamp != nil
}

func KubecConfigSecretName(customResource v1alpha1.App) string {
	return customResource.Spec.KubeConfig.Secret.Name
}

func KubecConfigSecretNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.KubeConfig.Secret.Namespace
}

func ReleaseName(customResource v1alpha1.App) string {
	return customResource.Spec.Name
}

// UserConfigMapName returns the name of the configmap that stores user level
// config for the provided app CR.
func UserConfigMapName(customResource v1alpha1.App) string {
	return customResource.Spec.UserConfig.ConfigMap.Name
}

// UserConfigMapNamespace returns the namespace of the configmap that stores user
// level config for the provided app CR.
func UserConfigMapNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.UserConfig.ConfigMap.Namespace
}

// UserSecretName returns the name of the secret that stores user level
// secrets for the provided app CR.
func UserSecretName(customResource v1alpha1.App) string {
	return customResource.Spec.UserConfig.Secret.Name
}

// UserSecretNamespace returns the namespace of the secret that stores user
// level secrets for the provided app CR.
func UserSecretNamespace(customResource v1alpha1.App) string {
	return customResource.Spec.UserConfig.Secret.Namespace
}

// VersionLabel returns the label value to determine if the custom resource is
// supported by this version of the operatorkit resource.
func VersionLabel(customResource v1alpha1.App) string {
	if val, ok := customResource.ObjectMeta.Labels[label.AppOperatorVersion]; ok {
		return val
	} else {
		return ""
	}
}
