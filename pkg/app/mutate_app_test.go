package app

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/app-admission-controller/config"
	"github.com/giantswarm/app-admission-controller/pkg/mutator"
)

func Test_MutateApp(t *testing.T) {
	ctx := context.Background()

	eggs2Cluster1920 := capiv1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eggs2",
			Namespace: "org-giantswarm",
			Labels: map[string]string{
				"cluster-operator.giantswarm.io/version": "5.8.0",
				"cluster.x-k8s.io/cluster-name":          "eggs2",
				"giantswarm.io/cluster":                  "eggs2",
				"giantswarm.io/organization":             "giantswarm",
				"giantswarm.io/service-priority":         "medium",
				"odp/provider":                           "aws",
				"odp/region":                             "eu-west-1",
				// release version < 19.3.0 to avoid PSP removal patches
				"release.giantswarm.io/version": "19.2.0",
			},
		},
	}
	eggs2Cluster1930 := capiv1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eggs2",
			Namespace: "org-giantswarm",
			Labels: map[string]string{
				"cluster-operator.giantswarm.io/version": "5.8.0",
				"cluster.x-k8s.io/cluster-name":          "eggs2",
				"giantswarm.io/cluster":                  "eggs2",
				"giantswarm.io/organization":             "giantswarm",
				"giantswarm.io/service-priority":         "medium",
				"odp/provider":                           "aws",
				"odp/region":                             "eu-west-1",
				// release version >= 19.3.0 to trigger PSP removal patches
				"release.giantswarm.io/version": "19.3.0",
			},
		},
	}
	eggs2ClusterCapiPSPdisabled := capiv1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eggs2",
			Namespace: "org-giantswarm",
			Labels: map[string]string{
				"cluster-operator.giantswarm.io/version": "5.8.0",
				"cluster.x-k8s.io/cluster-name":          "eggs2",
				"giantswarm.io/cluster":                  "eggs2",
				"giantswarm.io/organization":             "giantswarm",
				"giantswarm.io/service-priority":         "medium",
				// The psp disabling label is on capi clusters.
				"policy.giantswarm.io/psp-status": "disabled",
			},
		},
	}

	eggs2ClusterCapiNoLabel := capiv1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eggs2",
			Namespace: "org-giantswarm",
			Labels: map[string]string{
				"cluster-operator.giantswarm.io/version": "5.8.0",
				"cluster.x-k8s.io/cluster-name":          "eggs2",
				"giantswarm.io/cluster":                  "eggs2",
				"giantswarm.io/organization":             "giantswarm",
				"giantswarm.io/service-priority":         "medium",
			},
		},
	}

	xyz12Cluster1920 := capiv1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "xyz12",
			Namespace: "org-giantswarm",
			Labels: map[string]string{
				"cluster-operator.giantswarm.io/version": "5.8.0",
				"cluster.x-k8s.io/cluster-name":          "xyz12",
				"giantswarm.io/cluster":                  "xyz12",
				"giantswarm.io/organization":             "giantswarm",
				"giantswarm.io/service-priority":         "medium",
				"odp/provider":                           "aws",
				"odp/region":                             "eu-west-1",
				// release version < 19.3.0 to avoid PSP removal patches
				"release.giantswarm.io/version": "19.2.0",
			},
		},
	}

	tests := []struct {
		name               string
		oldObj             v1alpha1.App
		obj                v1alpha1.App
		apps               []*v1alpha1.App
		configMaps         []*corev1.ConfigMap
		secrets            []*corev1.Secret
		clusters           []*capiv1beta1.Cluster
		provider           string
		operation          admissionv1.Operation
		expectedPatches    []mutator.PatchOperation
		expectedConfigMaps []*corev1.ConfigMap
		expectedErr        string
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
			clusters: []*capiv1beta1.Cluster{
				&eggs2Cluster1920,
			},
			provider:  "aws",
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd("/metadata/labels", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), "kiam"),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), "3.0.0"),
				mutator.PatchAdd("/spec/extraConfigs", []v1alpha1.AppExtraConfig{}),
				mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
					Kind:      "configMap",
					Name:      "eggs2-cluster-values",
					Namespace: "eggs2",
					Priority:  bottomPriority,
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
			provider: "aws",
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
				mutator.PatchAdd("/spec/extraConfigs", []v1alpha1.AppExtraConfig{}),
				mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
					Kind:      "configMap",
					Name:      "eggs2-cluster-values",
					Namespace: "eggs2",
					Priority:  bottomPriority,
				}),
			},
			provider: "aws",
		},
		{
			name:   "case 3: set version label only",
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
			provider: "aws",
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
			name:   "case 4: no config map patch if it doesn't exist",
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
			provider: "aws",
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("other-app-values", "eggs2"),
			},
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
			},
		},
		{
			name:   "case 5: replace version label when it has legacy value 1.0.0",
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
			provider: "aws",
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
			name:   "case 6: no patches with legacy value 1.0.0 and no chart-operator app",
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
			provider:        "aws",
			apps:            nil,
			operation:       admissionv1.Create,
			expectedPatches: nil,
		},
		{
			name:   "case 7: flawless flow for org-namespaced app",
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
			clusters: []*capiv1beta1.Cluster{
				&eggs2Cluster1920,
			},
			provider:  "aws",
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), "kiam"),
				mutator.PatchAdd("/spec/extraConfigs", []v1alpha1.AppExtraConfig{}),
				mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
					Kind:      "configMap",
					Name:      "eggs2-cluster-values",
					Namespace: "org-eggs2",
					Priority:  bottomPriority,
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
			name:   "case 8: missing cluster label",
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
			provider:  "aws",
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd("/metadata/labels", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), "kiam"),
			},
		},
		{
			name:   "case 9: flawless flow for app in Release >= v19.3.0",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
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
			apps: []*v1alpha1.App{
				newTestApp("chart-operator", "eggs2", "3.0.0"),
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("eggs2-cluster-values", "eggs2"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("eggs2-kubeconfig", "eggs2"),
			},
			clusters: []*capiv1beta1.Cluster{
				&eggs2Cluster1930,
				&xyz12Cluster1920,
			},
			provider:  "aws",
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), "kiam"),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), "3.0.0"),
				mutator.PatchAdd("/spec/extraConfigs", []v1alpha1.AppExtraConfig{}),
				mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
					Kind:      "configMap",
					Name:      "eggs2-cluster-values",
					Namespace: "eggs2",
					Priority:  bottomPriority,
				}),
				mutator.PatchAdd("/metadata/labels/policy.giantswarm.io~1psp-status", "disabled"),
				mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
					Kind:      "configMap",
					Name:      "psp-removal-patch",
					Namespace: "eggs2",
					Priority:  150,
				}),
				mutator.PatchAdd("/spec/kubeConfig/context", map[string]string{
					"name": "eggs2",
				}),
				mutator.PatchAdd("/spec/kubeConfig/secret", map[string]string{
					"namespace": "eggs2",
					"name":      "eggs2-kubeconfig",
				}),
			},
			expectedConfigMaps: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "eggs2",
						Name:      "psp-removal-patch",
					},
					Data: map[string]string{"values": "global:\n  podSecurityStandards:\n    enforced: true"},
				},
			},
		},
		{
			name:   "case 10: no change flow for app in Release >= 19.3.0",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
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
					ExtraConfigs: []v1alpha1.AppExtraConfig{
						{
							Kind:      "configMap",
							Name:      "psp-removal-patch",
							Namespace: "eggs2",
							Priority:  150,
						},
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
			clusters: []*capiv1beta1.Cluster{
				&xyz12Cluster1920,
				&eggs2Cluster1930,
			},
			provider:  "aws",
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), "kiam"),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), "3.0.0"),
				mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
					Kind:      "configMap",
					Name:      "eggs2-cluster-values",
					Namespace: "eggs2",
					Priority:  bottomPriority,
				}),
				mutator.PatchAdd("/metadata/labels/policy.giantswarm.io~1psp-status", "disabled"),
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
			name:   "case 11: error when parent Cluster is missing",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
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
			apps: []*v1alpha1.App{
				newTestApp("chart-operator", "eggs2", "3.0.0"),
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("eggs2-cluster-values", "eggs2"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("eggs2-kubeconfig", "eggs2"),
			},
			clusters:    []*capiv1beta1.Cluster{},
			provider:    "aws",
			operation:   admissionv1.Create,
			expectedErr: "psp removal error: could not find a Cluster CR matching \"eggs2\" among 0 CRs",
		},
		{
			name:   "case 12: flawless flow for app in Release >= v19.3.0 with custom patch",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus-meta-operator",
					Namespace: "eggs2",
					Labels: map[string]string{
						label.Cluster: "eggs2",
					},
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "giantswarm",
					Name:      "prometheus-meta-operator",
					Namespace: "kube-system",
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						InCluster: false,
					},
					Version: "2.0.0",
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
			clusters: []*capiv1beta1.Cluster{
				&eggs2Cluster1930,
				&xyz12Cluster1920,
			},
			provider:  "aws",
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), "prometheus-meta-operator"),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), "3.0.0"),
				mutator.PatchAdd("/spec/extraConfigs", []v1alpha1.AppExtraConfig{}),
				mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
					Kind:      "configMap",
					Name:      "eggs2-cluster-values",
					Namespace: "eggs2",
					Priority:  bottomPriority,
				}),
				mutator.PatchAdd("/metadata/labels/policy.giantswarm.io~1psp-status", "disabled"),
				mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
					Kind:      "configMap",
					Name:      "psp-removal-patch-pmo",
					Namespace: "eggs2",
					Priority:  150,
				}),
				mutator.PatchAdd("/spec/kubeConfig/context", map[string]string{
					"name": "eggs2",
				}),
				mutator.PatchAdd("/spec/kubeConfig/secret", map[string]string{
					"namespace": "eggs2",
					"name":      "eggs2-kubeconfig",
				}),
			},
			expectedConfigMaps: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "eggs2",
						Name:      "psp-removal-patch-pmo",
					},
					Data: map[string]string{"values": "prometheus:\n  psp: false"},
				},
			},
		},
		{
			name:   "case 13: flawless flow for app in CAPx cluster with the psp-status disabled label.",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
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
			apps: []*v1alpha1.App{
				newTestApp("chart-operator", "eggs2", "3.0.0"),
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("eggs2-cluster-values", "eggs2"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("eggs2-kubeconfig", "eggs2"),
			},
			clusters: []*capiv1beta1.Cluster{
				&eggs2ClusterCapiPSPdisabled,
			},
			provider:  "capz",
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), "kiam"),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), "3.0.0"),
				mutator.PatchAdd("/spec/extraConfigs", []v1alpha1.AppExtraConfig{}),
				mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
					Kind:      "configMap",
					Name:      "eggs2-cluster-values",
					Namespace: "eggs2",
					Priority:  bottomPriority,
				}),
				mutator.PatchAdd("/metadata/labels/policy.giantswarm.io~1psp-status", "disabled"),
				mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
					Kind:      "configMap",
					Name:      "psp-removal-patch",
					Namespace: "eggs2",
					Priority:  150,
				}),
				mutator.PatchAdd("/spec/kubeConfig/context", map[string]string{
					"name": "eggs2",
				}),
				mutator.PatchAdd("/spec/kubeConfig/secret", map[string]string{
					"namespace": "eggs2",
					"name":      "eggs2-kubeconfig",
				}),
			},
			expectedConfigMaps: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "eggs2",
						Name:      "psp-removal-patch",
					},
					Data: map[string]string{"values": "global:\n  podSecurityStandards:\n    enforced: true"},
				},
			},
		},
		{
			name:   "case 14: flow with CAPx cluster where Cluster CR is missing.",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
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
			apps: []*v1alpha1.App{
				newTestApp("chart-operator", "eggs2", "3.0.0"),
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("eggs2-cluster-values", "eggs2"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("eggs2-kubeconfig", "eggs2"),
			},
			provider:  "capz",
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), "kiam"),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), "3.0.0"),
				mutator.PatchAdd("/spec/extraConfigs", []v1alpha1.AppExtraConfig{}),
				mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
					Kind:      "configMap",
					Name:      "eggs2-cluster-values",
					Namespace: "eggs2",
					Priority:  bottomPriority,
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
			name:   "case 15: flow with CAPx cluster where the disable label is missing.",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
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
			apps: []*v1alpha1.App{
				newTestApp("chart-operator", "eggs2", "3.0.0"),
			},
			configMaps: []*corev1.ConfigMap{
				newTestConfigMap("eggs2-cluster-values", "eggs2"),
			},
			secrets: []*corev1.Secret{
				newTestSecret("eggs2-kubeconfig", "eggs2"),
			},
			clusters: []*capiv1beta1.Cluster{
				&eggs2ClusterCapiNoLabel,
			},
			provider:  "capz",
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), "kiam"),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), "3.0.0"),
				mutator.PatchAdd("/spec/extraConfigs", []v1alpha1.AppExtraConfig{}),
				mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
					Kind:      "configMap",
					Name:      "eggs2-cluster-values",
					Namespace: "eggs2",
					Priority:  bottomPriority,
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
			name:   "case 15: no change flow for app in Release >= 19.3.0 (PSP Removal patch is not the last one)",
			oldObj: v1alpha1.App{},
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kiam",
					Namespace: "eggs2",
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
					ExtraConfigs: []v1alpha1.AppExtraConfig{
						{
							Kind:      "configMap",
							Name:      "psp-removal-patch",
							Namespace: "eggs2",
							Priority:  150,
						},
						{
							Kind:      "configMap",
							Name:      "eggs2-dummy-config",
							Namespace: "eggs2",
							Priority:  100,
						},
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
			clusters: []*capiv1beta1.Cluster{
				&xyz12Cluster1920,
				&eggs2Cluster1930,
			},
			provider:  "aws",
			operation: admissionv1.Create,
			expectedPatches: []mutator.PatchOperation{
				mutator.PatchAdd("/metadata/annotations", map[string]string{}),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppKubernetesName)), "kiam"),
				mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", replaceToEscape(label.AppOperatorVersion)), "3.0.0"),
				mutator.PatchAdd("/spec/extraConfigs/-", v1alpha1.AppExtraConfig{
					Kind:      "configMap",
					Name:      "eggs2-cluster-values",
					Namespace: "eggs2",
					Priority:  bottomPriority,
				}),
				mutator.PatchAdd("/metadata/labels/policy.giantswarm.io~1psp-status", "disabled"),
				mutator.PatchAdd("/spec/kubeConfig/context", map[string]string{
					"name": "eggs2",
				}),
				mutator.PatchAdd("/spec/kubeConfig/secret", map[string]string{
					"namespace": "eggs2",
					"name":      "eggs2-kubeconfig",
				}),
			},
		},
	}

	appSchemeBuilder := runtime.SchemeBuilder(schemeBuilder{
		v1alpha1.AddToScheme,
		capiv1beta1.AddToScheme,
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

			for _, cluster := range tc.clusters {
				g8sObjs = append(g8sObjs, cluster)
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
				Provider:  tc.provider,
				ConfigPatches: []config.ConfigPatch{
					{
						AppName:         "prometheus-meta-operator",
						ConfigMapSuffix: "pmo",
						Values:          "prometheus:\n  psp: false",
					},
					{
						AppName: "hello-world-app",
						Values:  "hello:\n  psp_deploy: false",
					},
				},
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
			for _, expectedCM := range tc.expectedConfigMaps {
				gotCM, err := k8sClient.K8sClient().CoreV1().ConfigMaps(expectedCM.Namespace).Get(ctx, expectedCM.Name, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("missing expected ConfigMap %s/%s: %s", expectedCM.Namespace, expectedCM.Name, err.Error())
				}
				if !reflect.DeepEqual(expectedCM.Data, gotCM.Data) {
					t.Fatalf("want matching ConfigMap %s/%s data:\n %s", expectedCM.Namespace, expectedCM.Name, cmp.Diff(gotCM.Data, expectedCM.Data))
				}
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
