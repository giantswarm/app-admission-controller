package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v6/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/app-admission-controller/internal/recorder"
)

func Test_ValidateApp(t *testing.T) {
	tests := []struct {
		name        string
		obj         *admissionv1.AdmissionRequest
		catalogs    []*v1alpha1.Catalog
		configMaps  []*corev1.ConfigMap
		secrets     []*corev1.Secret
		expectedErr string
	}{
		{
			name: "flawless app validation",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "kiam",
    							"namespace": "eggs2",
    							"labels": {
									"app-operator.giantswarm.io/version": "5.5.0"
    							}
							},
							"spec": {
    							"catalog": "giantswarm",
    							"name": "kiam",
    							"namespace": "kube-system",
    							"config": {
									"configMap": {
										"name": "eggs2-cluster-values",
										"namespace": "eggs2"
									}
								},
    							"kubeConfig": {
									"context": {
										"name": "eggs2-kubeconfig"
									},
									"inCluster": false,
									"secret": {
										"name": "eggs2-kubeconfig",
										"namespace": "eggs2"
									}
								},
								"userConfig": {
									"configMap": {
										"name": "kiam-user-values",
										"namespace": "eggs2"
									}
								},
								"version": "1.4.0"
							}
						}
					`),
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("eggs2-cluster-values", "eggs2"),
				newTestConfigMap("kiam-user-values", "eggs2"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("eggs2-kubeconfig", "eggs2"),
			},
		},
		{
			name: "flawless org-namespaced app validation",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "kiam",
    							"namespace": "org-eggs2",
    							"labels": {
									"app-operator.giantswarm.io/version": "2.6.0",
									"giantswarm.io/cluster": "eggs2"
    							}
							},
							"spec": {
								"catalog": "giantswarm",
								"name": "kiam",
								"namespace": "kube-system",
								"kubeConfig": {
									"context": {
										"name": "eggs2-kubeconfig"
									},
									"inCluster": false,
									"secret": {
										"name": "eggs2-kubeconfig",
										"namespace": "org-eggs2"
									}
								},
								"version": "1.4.0"
							}
						}
					`),
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("eggs2-kubeconfig", "org-eggs2"),
			},
		},
		{
			name: "missing cluster label",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "kiam",
    							"namespace": "org-eggs2",
    							"labels": {
									"app-operator.giantswarm.io/version": "2.6.0"
    							}
							},
							"spec": {
    							"catalog": "giantswarm",
    							"name": "kiam",
    							"namespace": "kube-system",
    							"kubeConfig": {
									"context": {
										"name": "eggs2-kubeconfig"
									},
									"inCluster": false,
									"secret": {
										"name": "eggs2-kubeconfig",
										"namespace": "org-eggs2"
									}
								},
								"version": "1.4.0"
							}
						}
					`),
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("eggs2-kubeconfig", "org-eggs2"),
			},
			expectedErr: "validation error: label `giantswarm.io/cluster` not found",
		},
		{
			// This test relates to the case 9 from mutation tests. Upon missing `giantswarm.io/cluster`
			// label mutation will produce an App CR with empty `.spec.kubeConfig`, before
			// validating configs, we should return error on missing label first.
			name: "missing cluster after mutation",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "kiam",
    							"namespace": "org-eggs2"
							},
							"spec": {
    							"catalog": "giantswarm",
    							"name": "kiam",
    							"namespace": "kube-system",
    							"kubeConfig": {
									"inCluster": false
								},
								"version": "1.4.0"
							}
						}
					`),
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("eggs2-kubeconfig", "org-eggs2"),
			},
			expectedErr: "validation error: label `giantswarm.io/cluster` not found",
		},
		{
			name: "flawless unique app validation",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "hello-world",
    							"namespace": "demo0",
    							"labels": {
									"app-operator.giantswarm.io/version": "0.0.0"
    							}
							},
							"spec": {
    							"catalog": "giantswarm",
    							"name": "hello-world",
    							"namespace": "demo0",
    							"config": {
									"configMap": {
										"name": "demo0-cluster-values",
										"namespace": "demo0"
									}
								},
    							"kubeConfig": {
									"inCluster": true
								},
								"userConfig": {
									"configMap": {
										"name": "hello-world-user-values",
										"namespace": "demo0"
									}
								},
								"version": "0.3.0"
							}
						}
					`),
				},
				UserInfo: authv1.UserInfo{
					Username: "00001@customer.onmicrosoft.com",
					Groups: []string{
						"customer:CUSTOM-GROUP",
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("demo0-cluster-values", "demo0"),
				newTestConfigMap("hello-world-user-values", "demo0"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("demo0-kubeconfig", "demo0"),
			},
		},
		{
			name: "referencing protected configuration as regular user",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "hello-world",
    							"namespace": "demo0",
    							"labels": {
									"app-operator.giantswarm.io/version": "0.0.0"
    							}
							},
							"spec": {
    							"catalog": "giantswarm",
    							"name": "hello-world",
    							"namespace": "demo0",
    							"config": {
									"configMap": {
										"name": "demo0-cluster-values",
										"namespace": "demo0"
									}
								},
    							"kubeConfig": {
									"context": {
										"name": "demo0-kubeconfig"
									},
									"inCluster": false,
									"secret": {
										"name": "demo0-kubeconfig",
										"namespace": "demo0"
									}
								},
								"userConfig": {
									"secret": {
										"name": "vault-token",
										"namespace": "giantswarm"
									}
								},
								"version": "0.3.0"
							}
						}
					`),
				},
				UserInfo: authv1.UserInfo{
					Username: "00001@customer.onmicrosoft.com",
					Groups: []string{
						"customer:CUSTOMER-GROUP",
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("demo0-cluster-values", "demo0"),
				newTestConfigMap("hello-world-user-values", "demo0"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("demo0-kubeconfig", "demo0"),
				newTestSecret("vault-token", "giantswarm"),
			},
			expectedErr: "validation error: references to giantswarm namespace not allowed for `0.0.0` labeld apps",
		},
		{
			name: "referencing protected configuration as privileged service account",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "app-operator-demo0",
    							"namespace": "demo0",
    							"labels": {
									"app-operator.giantswarm.io/version": "0.0.0"
    							}
							},
							"spec": {
    							"catalog": "control-plane",
    							"name": "app-operator",
    							"namespace": "demo0",
    							"config": {
									"configMap": {
										"name": "app-operator-cluster-values",
										"namespace": "giantswarm"
									}
								},
    							"kubeConfig": {
									"inCluster": true
								},
								"version": "0.3.0"
							}
						}
					`),
				},
				UserInfo: authv1.UserInfo{
					Username: "system:serviceaccount:giantswarm:cluster-operator-3-13-0",
					Groups: []string{
						"system:authenticated",
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("control-plane", "giantswarm"),
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("app-operator-cluster-values", "giantswarm"),
			},
		},
		{
			name: "modify existing App CR as a regular user",
			obj: &admissionv1.AdmissionRequest{
				Operation: "UPDATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "app-operator-demo0",
    							"namespace": "demo0",
    							"labels": {
									"app-operator.giantswarm.io/version": "0.0.0"
    							}
							},
							"spec": {
    							"catalog": "control-plane",
    							"name": "app-operator",
    							"namespace": "demo0",
    							"config": {
									"configMap": {
										"name": "app-operator-cluster-values",
										"namespace": "giantswarm"
									}
								},
    							"kubeConfig": {
									"inCluster": true
								},
								"userConfig": {
									"secret": {
										"name": "vault-token",
										"namespace": "giantswarm"
									}
								},
								"version": "0.3.0"
							}
						}
					`),
				},
				UserInfo: authv1.UserInfo{
					Username: "00001@customer.onmicrosoft.com",
					Groups: []string{
						"customer:CUSTOMER-GROUP",
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("control-plane", "giantswarm"),
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("app-operator-cluster-values", "giantswarm"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("vault-token", "giantswarm"),
			},
			expectedErr: "validation error: references to giantswarm namespace not allowed for `0.0.0` labeld apps",
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			t.Log(fmt.Sprintf("case %d: %s", i, tc.name))

			g8sObjs := make([]runtime.Object, 0)
			for _, cat := range tc.catalogs {
				g8sObjs = append(g8sObjs, cat)
			}

			k8sObjs := make([]runtime.Object, 0)
			for _, cm := range tc.configMaps {
				k8sObjs = append(k8sObjs, cm)
			}

			for _, secret := range tc.secrets {
				k8sObjs = append(k8sObjs, secret)
			}

			scheme := runtime.NewScheme()
			_ = v1alpha1.AddToScheme(scheme)

			k8sClient := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
				CtrlClient: fake.NewClientBuilder().
					WithScheme(scheme).
					WithRuntimeObjects(g8sObjs...).
					Build(),
				K8sClient: clientgofake.NewSimpleClientset(k8sObjs...),
			})

			var event recorder.Interface
			{
				c := recorder.Config{
					K8sClient: k8sClient,
					Component: "app-admission-controller",
				}

				event = recorder.New(c)
			}

			c := ValidatorConfig{
				Event:     event,
				K8sClient: k8sClient,
				Logger:    microloggertest.New(),
				Provider:  "aws",
			}

			r, err := NewValidator(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			_, err = r.Validate(tc.obj)
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
		})
	}
}

func newTestCatalog(name, namespace string) *v1alpha1.Catalog {
	return &v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.CatalogSpec{
			Description: name,
			Title:       name,
		},
	}
}
