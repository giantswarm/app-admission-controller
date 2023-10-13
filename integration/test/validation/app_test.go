//go:build k8srequired
// +build k8srequired

package validation

import (
	"context"
	"testing"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-admission-controller/integration/helpers"
)

const (
	appName       = "dex-app"
	catalog       = "control-plane-catalog"
	configMapName = "dex-config"
	namespace     = "test"
)

// TestFailWhenCatalogNotFound tests that the app CR is rejected if the
// referenced appcatalog CR does not exist.
func TestFailWhenCatalogNotFound(t *testing.T) {
	ctx := context.Background()

	var err error

	config := helpers.AppConfig{
		AppCatalog: "missing",
		AppLabels: map[string]string{
			label.AppOperatorVersion: "0.0.0",
			label.Cluster:            "xyz12",
		},
		AppName:         appName,
		AppNamespace:    "giantswarm",
		AppVersion:      "1.2.2",
		InCluster:       true,
		TargetCluster:   namespace,
		TargetNamespace: "giantswarm",
	}

	expectedError := "validation error: catalog `missing` not found"

	err = executeAppTest(ctx, expectedError, config)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}
}

// TestFailWhenClusterLabelNotFound tests that the app CR is rejected if the
// `giantswarm.io/cluster` label is not set.
func TestFailWhenClusterLabelNotFound(t *testing.T) {
	const (
		orgNamespace        = "org-acme"
		orgKubeConfigSecret = "test-kubeconfig"
	)

	ctx := context.Background()

	var err error

	err = config.CreateNamespace(ctx, orgNamespace)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	err = config.CreateSecret(ctx, orgKubeConfigSecret, orgNamespace)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	appConfig := helpers.AppConfig{
		AppCatalog:      catalog,
		AppName:         appName,
		AppNamespace:    orgNamespace,
		AppVersion:      "1.2.2",
		TargetCluster:   namespace,
		TargetNamespace: "default",
	}

	expectedError := "validation error: label `giantswarm.io/cluster` not found"

	err = executeAppTest(ctx, expectedError, appConfig)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}
}

// TestFailWhenTargetNamespaceNotAllowed tests that the app CR is rejected when
// user targets not allowed namespace.
func TestFailWhenTargetNamespaceNotAllowed(t *testing.T) {
	const (
		orgNamespace = "org-acme"
	)

	ctx := context.Background()

	var err error

	err = config.CreateNamespace(ctx, orgNamespace)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	appConfig := helpers.AppConfig{
		AppCatalog: catalog,
		AppLabels: map[string]string{
			label.AppOperatorVersion: "0.0.0",
			label.Cluster:            "xyz12",
		},
		AppName:         appName,
		AppNamespace:    orgNamespace,
		AppVersion:      "1.2.2",
		InCluster:       true,
		TargetNamespace: "kube-system",
	}
	expectedError := "validation error: target namespace kube-system is not allowed for in-cluster apps"

	err = executeAppTest(ctx, expectedError, appConfig)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}
}

// TestSkipValidationOnNamespaceDeletion tests that when the namespace
// containing an app CR is deleted the validation logic is skipped. This is
// done by checking if the app CR has a deletion timestamp.
func TestSkipValidationOnNamespaceDeletion(t *testing.T) {
	ctx := context.Background()

	var err error

	config.Logger.Debugf(ctx, "creating test resources in %#q namespace", namespace)

	err = createTestResources(ctx)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	config.Logger.Debugf(ctx, "created test resources in %#q namespace", namespace)

	config.Logger.Debugf(ctx, "deleting %#q namespace", namespace)

	err = config.AppTest.K8sClient().CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	config.Logger.Debugf(ctx, "deleted %#q namespace", namespace)

	config.Logger.Debugf(ctx, "waiting for %#q app deletion", appName)

	_, err = config.GetApp(ctx, appName, namespace)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	config.Logger.Debugf(ctx, "waited for %#q app deletion", appName)
}

func createTestResources(ctx context.Context) error {
	var err error

	err = config.CreateNamespace(ctx, namespace)
	if err != nil {
		return microerror.Mask(err)
	}

	err = config.CreateConfigMap(ctx, configMapName, namespace)
	if err != nil {
		return microerror.Mask(err)
	}

	err = config.CreateConfigMap(ctx, "test", namespace)
	if err != nil {
		return microerror.Mask(err)
	}

	err = config.CreateCatalog(ctx, catalog)
	if err != nil {
		return microerror.Mask(err)
	}

	appConfig := helpers.AppConfig{
		AppCatalog: catalog,
		AppLabels: map[string]string{
			label.AppOperatorVersion: "0.0.0",
			label.Cluster:            "xyz12",
		},
		AppName:         appName,
		AppNamespace:    namespace,
		AppVersion:      "1.2.2",
		ConfigName:      configMapName,
		InCluster:       true,
		TargetNamespace: namespace,
	}

	err = config.CreateApp(ctx, appConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
