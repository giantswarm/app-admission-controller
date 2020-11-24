package app

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned/fake"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/app-admission-controller/pkg/mutator"
)

func Test_MutateApp(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		obj             v1alpha1.App
		apps            []*v1alpha1.App
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
						InCluster: false,
					},
					Version: "1.4.0",
				},
			},
			apps: []*v1alpha1.App{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "chart-operator",
						Namespace: "eggs2",
						Labels: map[string]string{
							label.AppOperatorVersion: "3.0.0",
						},
					},
				},
			},
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/labels", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), "kiam"),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), "3.0.0"),
				mutator.PatchAdd("/spec/config", map[string]string{}),
				mutator.PatchAdd("/spec/config/configMap", map[string]string{
					"namespace": "eggs2",
					"name":      "eggs2-cluster-values",
				}),
				mutator.PatchAdd("/spec/kubeConfig/context", map[string]string{
					"name": "eggs2",
				}),
				mutator.PatchAdd("/spec/kubeConfig/secret", map[string]string{
					"namespace": "eggs2",
					"name":      "eggs2-kubeconfig",
				}),
			},
		},
		{
			name: "case 1: no patches",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
					Labels: map[string]string{
						"app.kubernetes.io/name": "kiam",
						label.AppOperatorVersion: "3.0.0",
					},
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
						Context: v1alpha1.AppSpecKubeConfigContext{
							Name: "eggs2",
						},
						InCluster: false,
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Namespace: "eggs2",
							Name:      "eggs2-kubeconfig",
						},
					},
					Version: "1.4.0",
				},
			},
		},
		{
			name: "case 2: cluster secret set",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
					Labels: map[string]string{
						"app.kubernetes.io/name": "kiam",
						label.AppOperatorVersion: "3.0.0",
					},
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
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/spec/config/configMap", map[string]string{
					"namespace": "eggs2",
					"name":      "eggs2-cluster-values",
				}),
			},
		},
		{
			name: "case 3: different configmap for nginx-ingress-controller-app",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nginx-ingress-controller-app",
					Namespace: "eggs2",
					Labels: map[string]string{
						"app.kubernetes.io/name": "kiam",
						label.AppOperatorVersion: "3.0.0",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "nginx-ingress-controller-app",
					Namespace: "kube-system",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						InCluster: true,
					},
					Version: "1.4.0",
				},
			},
			apps: []*v1alpha1.App{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "chart-operator",
						Namespace: "eggs2",
						Labels: map[string]string{
							label.AppOperatorVersion: "3.0.0",
						},
					},
				},
			},
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/spec/config", map[string]string{}),
				mutator.PatchAdd("/spec/config/configMap", map[string]string{
					"namespace": "eggs2",
					"name":      "ingress-controller-values",
				}),
			},
		},
		{
			name: "case 4: set version label only",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
					Labels: map[string]string{
						"app": "kiam",
					},
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
			apps: []*v1alpha1.App{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "chart-operator",
						Namespace: "eggs2",
						Labels: map[string]string{
							label.AppOperatorVersion: "3.0.0",
						},
					},
				},
			},
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), "3.0.0"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0)
			for _, app := range tc.apps {
				objs = append(objs, app)
			}

			k8sClient := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
				G8sClient: fake.NewSimpleClientset(objs...),
			})

			c := MutatorConfig{
				K8sClient: k8sClient,
				Logger:    microloggertest.New(),
			}
			r, err := NewMutator(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			patches, err := r.MutateApp(ctx, tc.obj)
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
