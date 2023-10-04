package app

import (
	"context"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app-admission-controller/pkg/mutator"
	"github.com/giantswarm/app-admission-controller/pkg/project"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// pssCutoffVersion represents first & lowest Giant Swarm Release version
	// which does not support PodSecurityPolicies.
	pssCutoffVersion, _ = semver.NewVersion("v19.2.0")
	clusterGVK          = schema.GroupVersionResource{
		Group:    "cluster.x-k8s.io",
		Version:  "v1beta1",
		Resource: "clusters",
	}
)

const (
	extraConfigName   = "pss-compliance-patch"
	extraConfigValues = `global:
  podsecuritystandards:
    enforced: true`
	topPriority = 150
)

// mutateConfigForPSSCompliance is a temporary solution to
// https://github.com/giantswarm/roadmap/issues/2716. Revert once migration to
// Release >= v19.2.0 is complete and managed apps no longer rely on PSPs.
func (m *Mutator) mutateConfigForPSSCompliance(ctx context.Context, app v1alpha1.App, result []mutator.PatchOperation) ([]mutator.PatchOperation, error) {
	clusterID := key.ClusterLabel(app)
	if clusterID == "" {
		// This App CR does not belong to any Workload Cluster - it does not need any more patches.
		return result, nil
	}

	extraConfig := v1alpha1.AppExtraConfig{
		Kind:      "configMap",
		Name:      extraConfigName,
		Namespace: app.Namespace,
		Priority:  topPriority,
	}

	// If extraConfigs are already patched with 'extraConfigName', let's save
	// ourselves some checks, ensure ConfigMap, and assume everything is in
	// order.
	ec := key.ExtraConfigs(app)
	if len(ec) > 0 && ec[len(ec)-1] == extraConfig {
		if err := m.ensureConfigMap(ctx, app.Namespace); err != nil {
			return nil, microerror.Mask(err)
		}
		return result, nil
	}

	// This App belongs to a Workload Cluster, which is using a certain Release
	// version. Let's determine what it is.
	var releaseVersion *semver.Version
	{
		// We don't want to guess Cluster's namespace because it's been
		// historically difficult. Cluster ID/name is unique, so we are relying
		// on that.
		clusterCRList := capiv1beta1.ClusterList{}
		err := m.k8sClient.CtrlClient().List(ctx, &clusterCRList, &client.ListOptions{})
		if err != nil {
			return nil, microerror.Maskf(pssComplianceError, "error listing Clusters: %v", err)
		}

		if len(clusterCRList.Items) != 1 {
			return nil, microerror.Maskf(pssComplianceError, "could not find one Cluster CR matching %q, found %d", clusterID, len(clusterCRList.Items))
		}
		clusterCR := clusterCRList.Items[0]

		label, ok := clusterCR.Labels[label.ReleaseVersion]
		if !ok {
			return nil, microerror.Maskf(pssComplianceError, "error infering Release version for Cluster %q", clusterID)
		}

		releaseSemver, err := semver.NewVersion(label)
		if err != nil {
			return nil, microerror.Maskf(pssComplianceError, "error parsing Release version %q as semver: %v", label, err)
		}

		releaseVersion = releaseSemver
	}

	if releaseVersion.LessThan(pssCutoffVersion) {
		// releaseVersion is lower than pssCutoffVersion and still supports PSS. Nothing to do.
		return result, nil
	}

	// We need to ensure configMap disabling PSPs exists and is added to
	// .spec.extraConfigs with highest priority.
	// Let's ensure the ConfigMap exists first...
	if err := m.ensureConfigMap(ctx, app.Namespace); err != nil {
		return nil, microerror.Mask(err)
	}

	// and add it to the list of extra configs in the App CR.
	result = append(result, mutator.PatchAdd("/spec/extraConfigs/-", extraConfig))

	return result, nil
}

// ensureConfigMap tries to create given ConfigMap. If it already exists, it
// updates the CM to ensure content consistency.
func (m *Mutator) ensureConfigMap(ctx context.Context, namespace string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      extraConfigName,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Data: map[string]string{
			"values": extraConfigValues,
		},
	}

	_, err := m.k8sClient.K8sClient().CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		_, err = m.k8sClient.K8sClient().CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	} else if err != nil {
		return microerror.Mask(err)
	}
	return nil
}