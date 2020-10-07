package unittest

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v2/pkg/label"
)

func DefaultApp() v1alpha1.App {
	cr := v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-control-plane-app",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				label.AppOperatorVersion: "0.0.0",
			},
		},
		Spec: v1alpha1.AppSpec{
			Name:      "my-control-plane-app",
			Namespace: "monitoring",
			Version:   "1.0.0",
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				InCluster: true,
			},
		},
	}

	return cr
}
