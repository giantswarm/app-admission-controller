package app

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-admission-controller/pkg/mutator"
)

func Test_MutateApp(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		obj             v1alpha1.App
		configMaps      []*corev1.ConfigMap
		secrets         []*corev1.Secret
		expectedPatches []mutator.PatchOperation
		expectedErr     string
	}{
		{
			name: "case 0: flawless flow",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "kiam",
					Namespace: "kube-system",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						InCluster: true,
					},
					Version: "1.4.0",
				},
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("eggs2-cluster-values", "eggs2"),
			},
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/spec/config", map[string]string{}),
				mutator.PatchAdd("/spec/config/configMap", map[string]string{}),
				mutator.PatchAdd("/spec/config/configMap/namespace", "eggs2"),
				mutator.PatchAdd("/spec/config/configMap/name", "eggs2-cluster-values"),
			},
		},
		{
			name: "case 1: no cluster configmap patches",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
				},
				Spec: v1alpha1.AppSpec{
					Catalog: "giantswarm",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Namespace: "eggs2",
							Name:      "eggs2-cluster-values",
						},
					},
					Name:      "kiam",
					Namespace: "kube-system",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						InCluster: true,
					},
					Version: "1.4.0",
				},
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("eggs2-cluster-values", "eggs2"),
			},
		},
		{
			name: "case 2: cluster secret set",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
				},
				Spec: v1alpha1.AppSpec{
					Catalog: "giantswarm",
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Namespace: "eggs2",
							Name:      "eggs2-cluster-secrets",
						},
					},
					Name:      "kiam",
					Namespace: "kube-system",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						InCluster: true,
					},
					Version: "1.4.0",
				},
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("eggs2-cluster-values", "eggs2"),
			},
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/spec/config/configMap", map[string]string{}),
				mutator.PatchAdd("/spec/config/configMap/namespace", "eggs2"),
				mutator.PatchAdd("/spec/config/configMap/name", "eggs2-cluster-values"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0)

			for _, cm := range tc.configMaps {
				objs = append(objs, cm)
			}

			for _, secret := range tc.secrets {
				objs = append(objs, secret)
			}

			k8sClient := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
				K8sClient: clientgofake.NewSimpleClientset(objs...),
			})

			c := MutatorConfig{
				K8sClient: k8sClient,
				Logger:    microloggertest.New(),
			}
			r, err := NewMutator(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			patches, err := r.MutateApp(ctx, tc.obj, tc.obj)
			switch {
			case err != nil && tc.expectedErr == "":
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.expectedErr != "":
				t.Fatalf("error == nil, want non-nil")
			}

			if err != nil && tc.expectedErr != "" {
				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Fatalf("error == %#v, want %#v ", err.Error(), tc.expectedErr)
				}
			}
			if !reflect.DeepEqual(patches, tc.expectedPatches) {
				t.Fatalf("want matching patches \n %s", cmp.Diff(patches, tc.expectedPatches))
			}
		})
	}
}

func newTestConfigMap(name, namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		Data: map[string]string{
			"values": "cluster: yaml\n",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
