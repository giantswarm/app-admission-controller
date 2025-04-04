package app

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/Masterminds/semver/v3"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v8/pkg/key"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-admission-controller/v2/config"
	"github.com/giantswarm/app-admission-controller/v2/pkg/mutator"
	"github.com/giantswarm/app-admission-controller/v2/pkg/project"
)

var (
	// pssCutoffVersion represents the first & lowest Giant Swarm Release
	// version which does not support PodSecurityPolicies.
	pssCutoffVersion, _ = semver.NewVersion("v19.3.0")
	// vintageProviders is a slice of provider names, like "aws";
	// mutateConfigForPSPRemoval is applied to vintage providers exclusively
	vintageProviders = []string{"aws", "azure", "kvm"}
	capiProviders    = []string{"capa", "capz", "cloud-director", "vsphere"}
)

const (
	defaultExtraConfigName   = "psp-removal-patch"
	defaultExtraConfigValues = `global:
  podSecurityStandards:
    enforced: true`
	bottomPriority = 1
	topPriority    = 150
	// pspLabel values have to match the ones defined in pss-operator.
	// See https://github.com/giantswarm/pss-operator/blob/main/service/controller/handler/pssversion/create.go#L25
	// pspLabelKeyForPatch has been escaped ('/' replaced with '~1') to fit JSONPatch format.
	pspLabelKeyForPatch = "policy.giantswarm.io~1psp-status"
	pspLabelKey         = "policy.giantswarm.io/psp-status"
	pspLabelVal         = "disabled"
)

// mutateConfigForPSPRemoval is a temporary solution to
// https://github.com/giantswarm/roadmap/issues/2716. Revert once migration to
// Release >= v19.3.0 is complete and managed apps no longer rely on PSPs.
func (m *Mutator) mutateConfigForPSPRemoval(ctx context.Context, app v1alpha1.App) ([]mutator.PatchOperation, error) {
	m.logger.Debugf(ctx, "App mutation for PSP Removal. App:%s, Namespace:%s\n",
		app.Name, app.Namespace)

	isVintageCluster := slices.Contains(vintageProviders, strings.ToLower(m.provider))
	isCAPICluster := slices.Contains(capiProviders, strings.ToLower(m.provider))

	if !isVintageCluster && !isCAPICluster {
		return nil, microerror.Maskf(pspRemovalError, "unsupported provider for PSP deprecation: %s", m.provider)
	}

	result := []mutator.PatchOperation{}

	clusterID := key.ClusterLabel(app)
	if clusterID == "" {
		// This App CR does not belong to any Workload Cluster - it does not
		// need any more patches.
		m.logger.Debugf(ctx, "App CR does not belong to any Workload Cluster. Skipping.\n")
		return result, nil
	}

	if app.Labels[label.AppOperatorVersion] == "0.0.0" && app.Namespace == "giantswarm" {
		m.logger.Debugf(ctx, "App is not WC app. Skipping.\n")
		// This App is not a Workload Cluster app, but has a ClusterID
		// annotation - it's an app bundle to be deployed to the MC.
		return result, nil
	}

	extraConfig := v1alpha1.AppExtraConfig{
		Kind:      "configMap",
		Name:      defaultExtraConfigName,
		Namespace: app.Namespace,
		Priority:  topPriority,
	}
	extraConfigName := defaultExtraConfigName
	extraConfigValues := defaultExtraConfigValues

	// If a custom patch is defined for this particular App name, override
	// extraConfig. Use a new name and custom values.
	ok, patch := m.appRequiresCustomPatch(ctx, app.Spec.Name)
	if ok {
		suffix := patch.ConfigMapSuffix
		if suffix == "" {
			suffix = patch.AppName
		}
		extraConfigName = fmt.Sprintf("%s-%s", defaultExtraConfigName, suffix)
		if len(extraConfigName) > 60 {
			extraConfigName = extraConfigName[:60]
		}
		extraConfig.Name = extraConfigName
		extraConfigValues = patch.Values
	}

	// If extraConfigs are already patched with 'extraConfigName', let's save
	// ourselves some checks, ensure ConfigMap, and assume everything is in
	// order.
	for _, ec := range key.ExtraConfigs(app) {
		if ec == extraConfig {
			// Ensure pssLabel to prevent any conflicts between pss-operator and other
			// operators, like Flux.
			result = append(result, mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", pspLabelKeyForPatch), pspLabelVal))

			if err := m.ensureConfigMap(ctx, app.Namespace, extraConfigName, extraConfigValues); err != nil {
				return nil, microerror.Mask(err)
			}
			m.logger.Debugf(ctx, "Extra config is already set. Skipping.\n")
			return result, nil
		}
	}

	// This App belongs to a Workload Cluster, which is using a certain Release
	// version. Let's determine what it is.

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
		if isVintageCluster {
			return nil, microerror.Maskf(pspRemovalError, "could not find a Cluster CR matching %q among %d CRs", clusterID, len(clusterCRList.Items))
		} else {
			// In CAPI clusters, Cluster CR can be created after the App CR.
			// pss-operator is responsible to trigger mutation of the App
			// once Cluster CR exists and has the psp label.
			m.logger.Debugf(ctx, "Could not find a Cluster CR, skipping and trust PSS-Operator.")
			return result, nil
		}
	}

	if isVintageCluster {
		m.logger.Debugf(ctx, "Vintage provider %s detected, checking release version\n", m.provider)
		var releaseVersion *semver.Version
		{
			label, ok := clusterCR.Labels[label.ReleaseVersion]
			if !ok {
				return nil, microerror.Maskf(pspRemovalError, "error inferring Release version for Cluster %q", clusterID)
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
	} else if isCAPICluster {
		m.logger.Debugf(ctx, "CAPI provider %s detected, checking cluster labels\n", m.provider)
		disableLabel, ok := clusterCR.Labels[pspLabelKey]
		// If the cluster CR does not have a label, we assume it still supports PSPs.
		if !ok {
			m.logger.Debugf(ctx, "Cluster doesn't have psp label. Skipping\n")
			return result, nil
		}
		if ok && disableLabel != pspLabelVal {
			return nil, microerror.Maskf(pspRemovalError, "cluster %q label found, but not set to %q", pspLabelKey, pspLabelVal)
		}
	}
	// Ensure pssLabel to prevent any conflicts between pss-operator and other
	// operators, like Flux.
	result = append(result, mutator.PatchAdd(fmt.Sprintf("/metadata/labels/%s", pspLabelKeyForPatch), pspLabelVal))

	// We need to ensure configMap disabling PSPs exists and is added to
	// .spec.extraConfigs with highest priority.
	// Let's ensure the ConfigMap exists first...
	if err := m.ensureConfigMap(ctx, app.Namespace, extraConfigName, extraConfigValues); err != nil {
		return nil, microerror.Mask(err)
	}

	result = append(result, mutator.PatchAdd("/spec/extraConfigs/-", extraConfig))

	m.logger.Debugf(ctx, "Mutation for PSPs are done.\n", m.provider)
	return result, nil
}

// ensureConfigMap tries to create given ConfigMap. If it already exists, it
// updates the CM to ensure content consistency.
func (m *Mutator) ensureConfigMap(ctx context.Context, namespace, name, values string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				label.ManagedBy: project.Name(),
			},
		},
		Data: map[string]string{
			"values": values,
		},
	}

	existing, err := m.k8sClient.K8sClient().CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		if reflect.DeepEqual(existing.Data, cm.Data) {
			m.logger.Debugf(ctx, "Configmap '%s' in '%s' namespace is up-to-date\n", name, namespace)
			return nil
		}
		m.logger.Debugf(ctx, "Updating configmap '%s' in '%s' namespace\n", name, namespace)
		_, err = m.k8sClient.K8sClient().CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	} else if apierrors.IsNotFound(err) {
		m.logger.Debugf(ctx, "Creating configmap '%s' in '%s' namespace\n", name, namespace)
		_, err = m.k8sClient.K8sClient().CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	}

	return err
}

// appRequiresCustomPatch checks if a particular app has a defined, customized
// extraConfig that prevents it from deploying PSPs. If it does, it returns the
// details, otherwise empty object.
func (m *Mutator) appRequiresCustomPatch(ctx context.Context, appSpecName string) (bool, config.ConfigPatch) {
	for _, patch := range m.configPatches {
		if patch.AppName == appSpecName {
			x := patch
			return true, x
		}
	}
	return false, config.ConfigPatch{}
}
