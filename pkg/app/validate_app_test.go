package app

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/micrologger/microloggertest"
	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/app-admission-controller/v2/internal/recorder"
	secins "github.com/giantswarm/app-admission-controller/v2/internal/security/inspector"
)

// This client has been added as a way to work around the error coming from here:
// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.16.3/pkg/client/fake/client.go#L595
//
// (ljakimczuk): Fake client of the Controller Runtime version we have been using so far (v0.6.5) did
// not support field selectors, hence our tests were working smooth, for such selectors were ignored.
// The new version supports them, but requires registering the appropriate indexing function, but
// apparently only selection by the '==' and '=' operators are allowed, and we use more, hence tests
// fail. Due to lack of other smart ideas, I decided to create this wrapper for fake client, which
// captures the List() call and prunes out unsupported selectors before calling the actual List() of
// the fake client.

type fakierClient struct {
	client.Client
}

func (m *fakierClient) List(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
	listOpts := client.ListOptions{}
	listOpts.ApplyOptions(opts)

	// create new selector by filtering out selections other than by the '==' or '=' operators
	newFieldSelectorsStr := []string{}
	if listOpts.FieldSelector != nil {
		for _, r := range listOpts.FieldSelector.Requirements() {
			if r.Operator == selection.Equals || r.Operator == selection.DoubleEquals {
				newFieldSelectorsStr = append(newFieldSelectorsStr, fmt.Sprintf("%s%s%s", r.Field, r.Operator, r.Value))
			}
		}
	}

	// parse the new selector
	newFieldSelectors, err := fields.ParseSelector(strings.Join(newFieldSelectorsStr, ","))
	if err != nil {
		return err
	}

	// do not pass empty string, for it leads to an error anyway
	if newFieldSelectors.Empty() {
		listOpts.FieldSelector = nil
	} else {
		listOpts.FieldSelector = newFieldSelectors
	}

	// run the actual List() of the fake client
	return m.Client.List(ctx, obj, &listOpts)
}

var appNameIndexer func(obj client.Object) []string = func(obj client.Object) []string {
	app, ok := obj.(*v1alpha1.App)
	if !ok {
		panic(fmt.Errorf("got %T object, want %T object", obj, v1alpha1.App{}))
	}

	return []string{app.Name}
}

func Test_ValidateApp(t *testing.T) {
	secInsCfg := secins.Config{
		Logger: microloggertest.New(),

		NamespaceBlacklist: []string{
			"capi-",
			"-prometheus",
			"draughtsman",
			"flux-giantswarm",
			"giantswarm",
			"kube-system",
			"monitoring",
			"org-giantswarm",
		},
		GroupWhitelist: []string{
			"giantswarm:giantswarm:giantswarm-admins",
		},
		UserWhitelist: []string{
			"system:serviceaccount:draughtsman:",
			"system:serviceaccount:giantswarm:",
			"system:serviceaccount:flux-giantswarm:",
			"system:serviceaccount:kube-system:",
		},
		AppBlacklist: []string{
			"app-operator",
		},
		CatalogBlacklist: []string{
			"control-plane-catalog",
			"control-plane-test-catalog",
		},
	}

	tests := []struct {
		name        string
		obj         *admissionv1.AdmissionRequest
		apps        []*v1alpha1.App
		aces        []*v1alpha1.AppCatalogEntry
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
			expectedErr: "security violation error: references to `giantswarm` namespace not allowed",
		},
		{
			name: "referencing protected configuration as regular service account",
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
									"configMap": {
										"name": "capi-credentials",
										"namespace": "capi-system"
									}
								},
								"version": "0.3.0"
							}
						}
					`),
				},
				UserInfo: authv1.UserInfo{
					Username: "system:serviceaccount:default:automation",
					Groups: []string{
						"system:authenticated",
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
			expectedErr: "security violation error: references to `capi-system` namespace not allowed",
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
			// App created by some GS member with access to the clusters, but being in the group
			// not explicitly listed in the user whitelists. Still, app created in private namespaces
			// means we pass the validation.
			name: "referencing protected kubeconfig from a private app",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "fake-controller",
    							"namespace": "org-giantswarm",
    							"labels": {
									"giantswarm.io/cluster": "gsgang",
									"app-operator.giantswarm.io/version": "0.0.0"
    							}
							},
							"spec": {
    							"catalog": "control-plane",
    							"name": "fake-controller",
    							"namespace": "giantswarm",
    							"kubeConfig": {
									"context": {
										"name": "gsgang-admin@gsgang"
									},
									"inCluster": false,
									"secret": {
										"name": "gsgang-kubeconfig",
										"namespace": "org-giantswarm"
									}
								},
								"version": "0.3.0"
							}
						}
					`),
				},
				UserInfo: authv1.UserInfo{
					Username: "user@giantswarm.io",
					Groups: []string{
						"customer:giantswarm:Employees",
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("control-plane", "giantswarm"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("gsgang-kubeconfig", "org-giantswarm"),
			},
		},
		{
			name: "referencing protected kubeconfig as a regular user",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "malicious-app",
    							"namespace": "default",
    							"labels": {
									"app-operator.giantswarm.io/version": "0.0.0"
    							}
							},
							"spec": {
    							"catalog": "custom",
    							"name": "malicious-app",
    							"namespace": "giantswarm",
    							"kubeConfig": {
									"context": {
										"name": "gsgang-admin@gsgang"
									},
									"inCluster": false,
									"secret": {
										"name": "gsgang-kubeconfig",
										"namespace": "org-giantswarm"
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
				newTestCatalog("custom", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("gsgang-kubeconfig", "org-giantswarm"),
			},
			expectedErr: "security violation error: references to `org-giantswarm` namespace not allowed",
		},
		{
			name: "install blacklisted app as regular service account",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "app-operator-new",
    							"namespace": "demo0",
    							"labels": {
									"app-operator.giantswarm.io/version": "0.0.0"
    							}
							},
							"spec": {
    							"catalog": "control-plane-catalog",
    							"name": "app-operator",
    							"namespace": "demo0",
    							"kubeConfig": {
									"inCluster": true
								},
								"version": "0.3.0"
							}
						}
					`),
				},
				UserInfo: authv1.UserInfo{
					Username: "system:serviceaccount:default:automation",
					Groups: []string{
						"system:authenticated",
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("control-plane", "giantswarm"),
			},
			expectedErr: "security violation error: installing `app-operator` from `control-plane-catalog` catalog is not allowed",
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
    							"catalog": "control-plane-catalog",
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
			expectedErr: "security violation error: installing `app-operator` from `control-plane-catalog` catalog is not allowed",
		},
		{
			name: "modify existing App CR as a GS member",
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
										"name": "app-operator-secrets",
										"namespace": "giantswarm"
									}
								},
								"version": "0.3.0"
							}
						}
					`),
				},
				OldObject: runtime.RawExtension{
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
					Username: "user@giantswarm.io",
					Groups: []string{
						"giantswarm:giantswarm:giantswarm-admins",
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
				newTestSecret("app-operator-secrets", "giantswarm"),
			},
		},
		{
			name: "cluster singletons in cluster namespace (app already present for the cluster)",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
		                {
		                    "apiVersion": "application.giantswarm.io/v1alpha1",
		                    "kind": "App",
		                    "metadata": {
		                        "name": "hello-world-second",
		                        "namespace": "demo01",
								"labels": {
									"app-operator.giantswarm.io/version": "6.0.0"
								}
		                    },
		                    "spec": {
		                        "catalog": "giantswarm",
		                        "name": "hello-world",
		                        "namespace": "hello-world",
								"kubeConfig": {
		                            "context": {
		                                "name": "demo01-kubeconfig"
		                            },
		                            "inCluster": false,
		                            "secret": {
		                                "name": "demo01-kubeconfig",
		                                "namespace": "demo01"
		                            }
		                        },
		                        "version": "0.3.0"
		                    }
		                }
		            `),
				},
				UserInfo: authv1.UserInfo{
					Username: "system:serviceaccount:default:automation",
					Groups: []string{
						"system:authenticated",
					},
				},
			},
			apps: []*v1alpha1.App{
				&v1alpha1.App{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hello-world-first",
						Namespace: "demo01",
						Labels: map[string]string{
							"app-operator.giantswarm.io/version": "6.0.0",
						},
					},
					Spec: v1alpha1.AppSpec{
						Catalog:   "giantswarm",
						Name:      "hello-world",
						Namespace: "hello-world",
						Version:   "0.3.0",
					},
				},
			},
			aces: []*v1alpha1.AppCatalogEntry{
				&v1alpha1.AppCatalogEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-hello-world-0.3.0",
						Namespace: "default",
					},
					Spec: v1alpha1.AppCatalogEntrySpec{
						Restrictions: &v1alpha1.AppCatalogEntrySpecRestrictions{
							ClusterSingleton: true,
						},
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("demo01-kubeconfig", "demo01"),
			},
			expectedErr: "app `hello-world` can only be installed once in cluster `demo01`",
		},
		{
			name: "cluster singletons in cluster namespace (app not yet configured for cluster)",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
		                {
		                    "apiVersion": "application.giantswarm.io/v1alpha1",
		                    "kind": "App",
		                    "metadata": {
		                        "name": "hello-world-demo01",
		                        "namespace": "demo01",
								"labels": {
									"app-operator.giantswarm.io/version": "6.0.0"
								}
		                    },
		                    "spec": {
		                        "catalog": "giantswarm",
		                        "name": "hello-world",
		                        "namespace": "hello-world",
		                        "kubeConfig": {
		                            "context": {
		                                "name": "demo01-kubeconfig"
		                            },
		                            "inCluster": false,
		                            "secret": {
		                                "name": "demo01-kubeconfig",
		                                "namespace": "demo01"
		                            }
		                        },
		                        "version": "0.3.0"
		                    }
		                }
		            `),
				},
				UserInfo: authv1.UserInfo{
					Username: "system:serviceaccount:default:automation",
					Groups: []string{
						"system:authenticated",
					},
				},
			},
			aces: []*v1alpha1.AppCatalogEntry{
				&v1alpha1.AppCatalogEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-hello-world-0.3.0",
						Namespace: "default",
					},
					Spec: v1alpha1.AppCatalogEntrySpec{
						Restrictions: &v1alpha1.AppCatalogEntrySpecRestrictions{
							ClusterSingleton: true,
						},
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("demo01-kubeconfig", "demo01"),
			},
		},
		{
			name: "namespace singletons in cluster namespace (app not yet configured for cluster)",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
		                {
		                    "apiVersion": "application.giantswarm.io/v1alpha1",
		                    "kind": "App",
		                    "metadata": {
		                        "name": "hello-world",
		                        "namespace": "demo01",
								"labels": {
 								   "app-operator.giantswarm.io/version": "6.0.0"
 							   }
		                    },
		                    "spec": {
		                        "catalog": "giantswarm",
		                        "name": "hello-world",
		                        "namespace": "hello-world",
		                        "kubeConfig": {
		                            "context": {
		                                "name": "demo01-kubeconfig"
		                            },
		                            "inCluster": false,
		                            "secret": {
		                                "name": "demo01-kubeconfig",
		                                "namespace": "demo01"
		                            }
		                        },
		                        "version": "0.3.0"
		                    }
		                }
		            `),
				},
				UserInfo: authv1.UserInfo{
					Username: "system:serviceaccount:default:automation",
					Groups: []string{
						"system:authenticated",
					},
				},
			},
			aces: []*v1alpha1.AppCatalogEntry{
				&v1alpha1.AppCatalogEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-hello-world-0.3.0",
						Namespace: "default",
					},
					Spec: v1alpha1.AppCatalogEntrySpec{
						Restrictions: &v1alpha1.AppCatalogEntrySpecRestrictions{
							NamespaceSingleton: true,
						},
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("demo01-kubeconfig", "demo01"),
			},
		},
		{
			name: "namespace singletons in cluster namespace (app configured already, but for different namespace)",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
		                {
		                    "apiVersion": "application.giantswarm.io/v1alpha1",
		                    "kind": "App",
		                    "metadata": {
		                        "name": "hello-world-second",
		                        "namespace": "demo01",
								"labels": {
									"app-operator.giantswarm.io/version": "6.0.0"
								}
		                    },
		                    "spec": {
		                        "catalog": "giantswarm",
		                        "name": "hello-world",
		                        "namespace": "hello-world-second",
		                        "kubeConfig": {
		                            "context": {
		                                "name": "demo01-kubeconfig"
		                            },
		                            "inCluster": false,
		                            "secret": {
		                                "name": "demo01-kubeconfig",
		                                "namespace": "demo01"
		                            }
		                        },
		                        "version": "0.3.0"
		                    }
		                }
		            `),
				},
				UserInfo: authv1.UserInfo{
					Username: "system:serviceaccount:default:automation",
					Groups: []string{
						"system:authenticated",
					},
				},
			},
			apps: []*v1alpha1.App{
				&v1alpha1.App{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hello-world-first",
						Namespace: "demo01",
					},
					Spec: v1alpha1.AppSpec{
						Catalog:   "giantswarm",
						Name:      "hello-world",
						Namespace: "hello-world-first",
						Version:   "0.3.0",
					},
				},
			},
			aces: []*v1alpha1.AppCatalogEntry{
				&v1alpha1.AppCatalogEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-hello-world-0.3.0",
						Namespace: "default",
					},
					Spec: v1alpha1.AppCatalogEntrySpec{
						Restrictions: &v1alpha1.AppCatalogEntrySpecRestrictions{
							NamespaceSingleton: true,
						},
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("demo01-kubeconfig", "demo01"),
			},
		},
		{
			name: "namespace singletons in cluster namespace (app configured already for the given namespace)",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
		                {
		                    "apiVersion": "application.giantswarm.io/v1alpha1",
		                    "kind": "App",
		                    "metadata": {
		                        "name": "hello-world-second",
		                        "namespace": "demo01",
								"labels": {
									"app-operator.giantswarm.io/version": "6.0.0"
								}
		                    },
		                    "spec": {
		                        "catalog": "giantswarm",
		                        "name": "hello-world",
		                        "namespace": "hello-world",
		                        "kubeConfig": {
		                            "context": {
		                                "name": "demo01-kubeconfig"
		                            },
		                            "inCluster": false,
		                            "secret": {
		                                "name": "demo01-kubeconfig",
		                                "namespace": "demo01"
		                            }
		                        },
		                        "version": "0.3.0"
		                    }
		                }
		            `),
				},
				UserInfo: authv1.UserInfo{
					Username: "system:serviceaccount:default:automation",
					Groups: []string{
						"system:authenticated",
					},
				},
			},
			apps: []*v1alpha1.App{
				&v1alpha1.App{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hello-world-first",
						Namespace: "demo01",
					},
					Spec: v1alpha1.AppSpec{
						Catalog:   "giantswarm",
						Name:      "hello-world",
						Namespace: "hello-world",
						Version:   "0.3.0",
					},
				},
			},
			aces: []*v1alpha1.AppCatalogEntry{
				&v1alpha1.AppCatalogEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-hello-world-0.3.0",
						Namespace: "default",
					},
					Spec: v1alpha1.AppCatalogEntrySpec{
						Restrictions: &v1alpha1.AppCatalogEntrySpecRestrictions{
							NamespaceSingleton: true,
						},
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("demo01-kubeconfig", "demo01"),
			},
			expectedErr: "app `hello-world` can only be installed only once in namespace `hello-world`",
		},
		{
			name: "cluster singletons in org namespace (app already present for the cluster)",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "hello-world-demo01",
    							"namespace": "org-test",
    							"labels": {
									"giantswarm.io/cluster": "demo01"
    							}
							},
							"spec": {
    							"catalog": "giantswarm",
    							"name": "hello-world",
    							"namespace": "hello-world",
    							"kubeConfig": {
									"context": {
										"name": "demo01-kubeconfig"
									},
									"inCluster": false,
									"secret": {
										"name": "demo01-kubeconfig",
										"namespace": "demo01"
									}
								},
								"version": "0.3.0"
							}
						}
					`),
				},
				UserInfo: authv1.UserInfo{
					Username: "system:serviceaccount:default:automation",
					Groups: []string{
						"system:authenticated",
					},
				},
			},
			apps: []*v1alpha1.App{
				&v1alpha1.App{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hello-world-demo01-existing",
						Namespace: "org-test",
						Labels: map[string]string{
							"giantswarm.io/cluster": "demo01",
						},
					},
					Spec: v1alpha1.AppSpec{
						Catalog:   "giantswarm",
						Name:      "hello-world",
						Namespace: "hello-world",
						Version:   "0.3.0",
					},
				},
			},
			aces: []*v1alpha1.AppCatalogEntry{
				&v1alpha1.AppCatalogEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-hello-world-0.3.0",
						Namespace: "default",
					},
					Spec: v1alpha1.AppCatalogEntrySpec{
						Restrictions: &v1alpha1.AppCatalogEntrySpecRestrictions{
							ClusterSingleton: true,
						},
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("demo01-kubeconfig", "demo01"),
			},
			expectedErr: "app `hello-world` can only be installed once in cluster `demo01`",
		},
		{
			name: "cluster singletons in org namespace (app not yet configured for cluster)",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "hello-world-demo01",
    							"namespace": "org-test",
    							"labels": {
									"giantswarm.io/cluster": "demo01"
    							}
							},
							"spec": {
    							"catalog": "giantswarm",
    							"name": "hello-world",
    							"namespace": "hello-world",
    							"kubeConfig": {
									"context": {
										"name": "demo01-kubeconfig"
									},
									"inCluster": false,
									"secret": {
										"name": "demo01-kubeconfig",
										"namespace": "demo01"
									}
								},
								"version": "0.3.0"
							}
						}
					`),
				},
				UserInfo: authv1.UserInfo{
					Username: "system:serviceaccount:default:automation",
					Groups: []string{
						"system:authenticated",
					},
				},
			},
			apps: []*v1alpha1.App{
				&v1alpha1.App{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hello-world-demo02",
						Namespace: "org-test",
						Labels: map[string]string{
							"giantswarm.io/cluster": "demo02",
						},
					},
					Spec: v1alpha1.AppSpec{
						Catalog:   "giantswarm",
						Name:      "hello-world",
						Namespace: "hello-world",
						Version:   "0.3.0",
					},
				},
			},
			aces: []*v1alpha1.AppCatalogEntry{
				&v1alpha1.AppCatalogEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-hello-world-0.3.0",
						Namespace: "default",
					},
					Spec: v1alpha1.AppCatalogEntrySpec{
						Restrictions: &v1alpha1.AppCatalogEntrySpecRestrictions{
							ClusterSingleton: true,
						},
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("demo01-kubeconfig", "demo01"),
			},
		},
		{
			name: "namespace singletons in org namespace (app configured for different clusters)",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "hello-world-demo01",
    							"namespace": "org-test",
    							"labels": {
									"giantswarm.io/cluster": "demo01"
    							}
							},
							"spec": {
    							"catalog": "giantswarm",
    							"name": "hello-world",
    							"namespace": "hello-world",
    							"kubeConfig": {
									"context": {
										"name": "demo01-kubeconfig"
									},
									"inCluster": false,
									"secret": {
										"name": "demo01-kubeconfig",
										"namespace": "demo01"
									}
								},
								"version": "0.3.0"
							}
						}
					`),
				},
				UserInfo: authv1.UserInfo{
					Username: "system:serviceaccount:default:automation",
					Groups: []string{
						"system:authenticated",
					},
				},
			},
			apps: []*v1alpha1.App{
				&v1alpha1.App{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hello-world-demo02",
						Namespace: "org-test",
						Labels: map[string]string{
							"giantswarm.io/cluster": "demo02",
						},
					},
					Spec: v1alpha1.AppSpec{
						Catalog:   "giantswarm",
						Name:      "hello-world",
						Namespace: "hello-world",
						Version:   "0.3.0",
					},
				},
			},
			aces: []*v1alpha1.AppCatalogEntry{
				&v1alpha1.AppCatalogEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-hello-world-0.3.0",
						Namespace: "default",
					},
					Spec: v1alpha1.AppCatalogEntrySpec{
						Restrictions: &v1alpha1.AppCatalogEntrySpecRestrictions{
							NamespaceSingleton: true,
						},
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("demo01-kubeconfig", "demo01"),
			},
		},
		{
			name: "namespace singletons in org namespace (app configured for the same cluster)",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "hello-world-demo01-second",
    							"namespace": "org-test",
    							"labels": {
									"giantswarm.io/cluster": "demo01"
    							}
							},
							"spec": {
    							"catalog": "giantswarm",
    							"name": "hello-world",
    							"namespace": "hello-world",
    							"kubeConfig": {
									"context": {
										"name": "demo01-kubeconfig"
									},
									"inCluster": false,
									"secret": {
										"name": "demo01-kubeconfig",
										"namespace": "demo01"
									}
								},
								"version": "0.3.0"
							}
						}
					`),
				},
				UserInfo: authv1.UserInfo{
					Username: "system:serviceaccount:default:automation",
					Groups: []string{
						"system:authenticated",
					},
				},
			},
			apps: []*v1alpha1.App{
				&v1alpha1.App{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hello-world-demo01-first",
						Namespace: "org-test",
						Labels: map[string]string{
							"giantswarm.io/cluster": "demo01",
						},
					},
					Spec: v1alpha1.AppSpec{
						Catalog:   "giantswarm",
						Name:      "hello-world",
						Namespace: "hello-world",
						Version:   "0.3.0",
					},
				},
			},
			aces: []*v1alpha1.AppCatalogEntry{
				&v1alpha1.AppCatalogEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-hello-world-0.3.0",
						Namespace: "default",
					},
					Spec: v1alpha1.AppCatalogEntrySpec{
						Restrictions: &v1alpha1.AppCatalogEntrySpecRestrictions{
							NamespaceSingleton: true,
						},
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("demo01-kubeconfig", "demo01"),
			},
			expectedErr: "app `hello-world` can only be installed only once in namespace `hello-world`",
		},
		{
			name: "namespace singletons in org namespace (app configured for the same cluster, different namespace)",
			obj: &admissionv1.AdmissionRequest{
				Operation: "CREATE",
				Object: runtime.RawExtension{
					Raw: []byte(`
						{
							"apiVersion": "application.giantswarm.io/v1alpha1",
							"kind": "App",
							"metadata": {
    							"name": "hello-world-demo01-second",
    							"namespace": "org-test",
    							"labels": {
									"giantswarm.io/cluster": "demo01"
    							}
							},
							"spec": {
    							"catalog": "giantswarm",
    							"name": "hello-world",
    							"namespace": "hello-world-second",
    							"kubeConfig": {
									"context": {
										"name": "demo01-kubeconfig"
									},
									"inCluster": false,
									"secret": {
										"name": "demo01-kubeconfig",
										"namespace": "demo01"
									}
								},
								"version": "0.3.0"
							}
						}
					`),
				},
				UserInfo: authv1.UserInfo{
					Username: "system:serviceaccount:default:automation",
					Groups: []string{
						"system:authenticated",
					},
				},
			},
			apps: []*v1alpha1.App{
				&v1alpha1.App{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hello-world-demo01-first",
						Namespace: "org-test",
						Labels: map[string]string{
							"giantswarm.io/cluster": "demo01",
						},
					},
					Spec: v1alpha1.AppSpec{
						Catalog:   "giantswarm",
						Name:      "hello-world",
						Namespace: "hello-world-first",
						Version:   "0.3.0",
					},
				},
			},
			aces: []*v1alpha1.AppCatalogEntry{
				&v1alpha1.AppCatalogEntry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giantswarm-hello-world-0.3.0",
						Namespace: "default",
					},
					Spec: v1alpha1.AppCatalogEntrySpec{
						Restrictions: &v1alpha1.AppCatalogEntrySpecRestrictions{
							NamespaceSingleton: true,
						},
					},
				},
			},
			catalogs: []*v1alpha1.Catalog{
				newTestCatalog("giantswarm", "default"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("demo01-kubeconfig", "demo01"),
			},
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			t.Logf("case %d: %s", i, tc.name)

			g8sObjs := make([]runtime.Object, 0)
			for _, cat := range tc.catalogs {
				g8sObjs = append(g8sObjs, cat)
			}
			for _, app := range tc.apps {
				g8sObjs = append(g8sObjs, app)
			}
			for _, ace := range tc.aces {
				g8sObjs = append(g8sObjs, ace)
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

			fakeCtrlClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(g8sObjs...).
				WithIndex(&v1alpha1.App{}, "metadata.name", appNameIndexer).
				Build()

			k8sClient := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
				CtrlClient: &fakierClient{fakeCtrlClient},
				K8sClient:  clientgofake.NewSimpleClientset(k8sObjs...),
			})

			var event recorder.Interface
			{
				c := recorder.Config{
					K8sClient: k8sClient,
					Component: "app-admission-controller",
				}

				event = recorder.New(c)
			}

			ins, err := secins.New(secInsCfg)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			c := ValidatorConfig{
				Event:     event,
				K8sClient: k8sClient,
				Logger:    microloggertest.New(),
				Provider:  "aws",
				Inspector: ins,
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
