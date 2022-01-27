//go:build k8srequired
// +build k8srequired

package helpers

import (
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppConfig struct {
	AppCatalog        string
	AppLabels         map[string]string
	AppName           string
	AppNamespace      string
	AppVersion        string
	ConfigName        string
	DefaultingEnabled bool
	InCluster         bool
	TargetCluster     string
	TargetNamespace   string
}

func GetAppCR(config AppConfig) *v1alpha1.App {
	app := &v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.AppName,
			Namespace: config.AppNamespace,
			Labels:    config.AppLabels,
		},
		Spec: v1alpha1.AppSpec{
			Catalog:   config.AppCatalog,
			Name:      config.AppName,
			Namespace: config.TargetNamespace,
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				InCluster: config.InCluster,
			},
			Version: config.AppVersion,
		},
	}

	if config.ConfigName != "" {
		app.Spec.Config = v1alpha1.AppSpecConfig{
			ConfigMap: v1alpha1.AppSpecConfigConfigMap{
				Name:      config.ConfigName,
				Namespace: config.AppNamespace,
			},
		}
	}

	if !config.InCluster && config.DefaultingEnabled {
		app.Spec.KubeConfig.Context = v1alpha1.AppSpecKubeConfigContext{
			Name: config.TargetCluster,
		}

		app.Spec.KubeConfig.Secret = v1alpha1.AppSpecKubeConfigSecret{
			Name:      config.TargetCluster + "-kubeconfig",
			Namespace: config.AppNamespace,
		}
	}

	return app
}
