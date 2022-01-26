package app

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v6/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/app-admission-controller/pkg/mutator"
)

func Test_MutateApp(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		oldObj          v1alpha1.App
		obj             v1alpha1.App
		apps            []*v1alpha1.App
		configMaps      []*corev1.ConfigMap
		secrets         []*corev1.Secret
		operation       admissionv1.Operation
		expectedPatches []mutator.PatchOperation
		expectedErr     string
	}{
		{
			name:   "case 0: flawless flow",
			oldObj: v1alpha1.App{},
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
				newTestApp("chart-operator", "eggs2", "3.0.0"),
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("eggs2-cluster-values", "eggs2"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("eggs2-kubeconfig", "eggs2"),
			},
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
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
			name:   "case 1: no patches",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
					Annotations: map[string]string{
						"some": "annotation",
					},
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
			name:   "case 2: cluster secret set",
			oldObj: v1alpha1.App{},
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
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("eggs2-cluster-values", "eggs2"),
			},
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd("/spec/config/configMap", map[string]string{
					"namespace": "eggs2",
					"name":      "eggs2-cluster-values",
				}),
			},
		},
		{
			name:   "case 3: different configmap for nginx-ingress-controller-app",
			oldObj: v1alpha1.App{},
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
				newTestApp("chart-operator", "eggs2", "3.0.0"),
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("ingress-controller-values", "eggs2"),
			},
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd("/spec/config", map[string]string{}),
				mutator.PatchAdd("/spec/config/configMap", map[string]string{
					"namespace": "eggs2",
					"name":      "ingress-controller-values",
				}),
			},
		},
		{
			name:   "case 4: set version label only",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
					Labels: map[string]string{
						"app": "kiam",
					},
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
			apps: []*v1alpha1.App{
				newTestApp("chart-operator", "eggs2", "3.1.0"),
			},
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), "3.1.0"),
			},
		},
		{
			name:   "case 5: no config map patch if it doesn't exist",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
					Labels: map[string]string{
						"app":                    "kiam",
						label.AppOperatorVersion: "3.0.0",
					},
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
				newTestConfigMap("other-app-values", "eggs2"),
			},
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
			},
		},
		{
			name:   "case 6: replace version label when it has legacy value 1.0.0",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
					Labels: map[string]string{
						"app":                    "kiam",
						label.AppOperatorVersion: "1.0.0",
					},
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
			apps: []*v1alpha1.App{
				newTestApp("chart-operator", "eggs2", "3.1.0"),
			},
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), "3.1.0"),
			},
		},
		{
			name:   "case 7: no patches with legacy value 1.0.0 and no chart-operator app",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
					Labels: map[string]string{
						"app":                    "kiam",
						label.AppOperatorVersion: "1.0.0",
					},
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
			apps:            nil,
			operation:       admissionv1.Create,
			expectedPatches: nil,
		},
		{
			name:   "case 8: flawless flow for org-namespaced app",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "org-eggs2",
					Labels: map[string]string{
						label.Cluster: "eggs2",
					},
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
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("eggs2-cluster-values", "org-eggs2"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("eggs2-kubeconfig", "org-eggs2"),
			},
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), "kiam"),
				mutator.PatchAdd("/spec/config", map[string]string{}),
				mutator.PatchAdd("/spec/config/configMap", map[string]string{
					"namespace": "org-eggs2",
					"name":      "eggs2-cluster-values",
				}),
				mutator.PatchAdd("/spec/kubeConfig/context", map[string]string{
					"name": "eggs2",
				}),
				mutator.PatchAdd("/spec/kubeConfig/secret", map[string]string{
					"namespace": "org-eggs2",
					"name":      "eggs2-kubeconfig",
				}),
			},
		},
		{
			// When `giantswarm.io/cluster` label is missing for the org-namespaced
			// apps, then some patches will be skipped due to mutator not being able
			// to determine correct config names. We then expect the validator to return
			// error upon spotting missing label.
			name:   "case 9: missing cluster label",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "org-eggs2",
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
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("eggs2-cluster-values", "org-eggs2"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("eggs2-kubeconfig", "org-eggs2"),
			},
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd("/metadata/labels", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), "kiam"),
			},
		},
	}

	appSchemeBuilder := runtime.SchemeBuilder(schemeBuilder{
		v1alpha1.AddToScheme,
	})
	err := appSchemeBuilder.AddToScheme(scheme.Scheme)
	if err != nil {
		t.Fatalf("error == %#v, want nil", err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Log(tc.name)

			g8sObjs := make([]runtime.Object, 0)

			for _, app := range tc.apps {
				g8sObjs = append(g8sObjs, app)
			}

			k8sObjs := make([]runtime.Object, 0)

			for _, cm := range tc.configMaps {
				k8sObjs = append(k8sObjs, cm)
			}

			for _, secret := range tc.secrets {
				k8sObjs = append(k8sObjs, secret)
			}

			k8sClient := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
				CtrlClient: fake.NewClientBuilder().
					WithScheme(scheme.Scheme).
					WithRuntimeObjects(g8sObjs...).
					Build(),
				K8sClient: clientgofake.NewSimpleClientset(k8sObjs...),
			})

			c := MutatorConfig{
				K8sClient: k8sClient,
				Logger:    microloggertest.New(),
			}
			r, err := NewMutator(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			patches, err := r.MutateApp(ctx, tc.oldObj, tc.obj, tc.operation)
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

func newTestApp(name, namespace, versionLabel string) *v1alpha1.App {
	return &v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				label.AppOperatorVersion: versionLabel,
			},
		},
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

func newTestSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		Data: map[string][]byte{
			"values": []byte("secret"),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

type schemeBuilder []func(*runtime.Scheme) error
