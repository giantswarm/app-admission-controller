package app

import (
	"context"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/app-admission-controller/pkg/mutator"
	"github.com/giantswarm/app-admission-controller/pkg/project"
)

var (
	// pssCutoffVersion represents the first & lowest Giant Swarm Release
	// version which does not support PodSecurityPolicies.
	pssCutoffVersion, _ = semver.NewVersion("v19.3.0")
	// vintageProviders is a slice of provider names, like "aws";
	// mutateConfigForPSPRemoval is applied to vintage providers exclusively
	vintageProviders = []string{"aws", "azure", "kvm"}
)

const (
	extraConfigName   = "psp-removal-patch"
	extraConfigValues = `global:
  podsecuritystandards:
    enforced: true`
	topPriority = 150
)

// mutateConfigForPSPRemoval is a temporary solution to
// https://github.com/giantswarm/roadmap/issues/2716. Revert once migration to
// Release >= v19.3.0 is complete and managed apps no longer rely on PSPs.
func (m *Mutator) mutateConfigForPSPRemoval(ctx context.Context, app v1alpha1.App) ([]mutator.PatchOperation, error) {
	result := []mutator.PatchOperation{}

	if !slices.Contains(vintageProviders, strings.ToLower(m.provider)) {
		// PSP patch is applicable to vintage providers only.
		return result, nil
	}

	clusterID := key.ClusterLabel(app)
	if clusterID == "" {
		// This App CR does not belong to any Workload Cluster - it does not
		// need any more patches.
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
			return nil, microerror.Maskf(pspRemovalError, "error listing Clusters: %v", err)
		}

		var clusterCR *capiv1beta1.Cluster
		for _, item := range clusterCRList.Items {
			if item.Name == clusterID {
				x := item
				clusterCR = &x
				break
			}
		}

		if clusterCR == nil {
			return nil, microerror.Maskf(pspRemovalError, "could not find a Cluster CR matching %q among %d CRs", clusterID, len(clusterCRList.Items))
		}

		label, ok := clusterCR.Labels[label.ReleaseVersion]
		if !ok {
			return nil, microerror.Maskf(pspRemovalError, "error infering Release version for Cluster %q", clusterID)
		}

		releaseSemver, err := semver.NewVersion(label)
		if err != nil {
			return nil, microerror.Maskf(pspRemovalError, "error parsing Release version %q as semver: %v", label, err)
		}

		releaseVersion = releaseSemver
	}

	if releaseVersion.LessThan(pssCutoffVersion) {
		// releaseVersion is lower than pssCutoffVersion and still supports PSPs. Nothing to do.
		return result, nil
	}

	// We need to ensure configMap disabling PSPs exists and is added to
	// .spec.extraConfigs with highest priority.
	// Let's ensure the ConfigMap exists first...
	if err := m.ensureConfigMap(ctx, app.Namespace); err != nil {
		return nil, microerror.Mask(err)
	}

	// and add it to the list of extra configs in the App CR.
	if len(key.ExtraConfigs(app)) == 0 {
		result = append(result, mutator.PatchAdd("/spec/extraConfigs", []v1alpha1.AppExtraConfig{}))
	}
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
