//go:build k8srequired
// +build k8srequired

package validation

import (
	"context"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	config := appConfig{
		appCatalog:      "missing",
		appLabels:       map[string]string{label.AppOperatorVersion: "3.0.0"},
		appName:         appName,
		appNamespace:    "giantswarm",
		appVersion:      "1.2.2",
		inCluster:       true,
		targetCluster:   namespace,
		targetNamespace: "giantswarm",
	}

	expectedError := "validation error: catalog `missing` not found"

	err = executeWithApp(ctx, expectedError, config)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}
}

// TestFailWhenClusterLabelNotFound tests that the app CR is rejected if the
// `giantswarm.io/cluster` label is not set.
func TestFailWhenClusterLabelNotFound(t *testing.T) {
	const (
		orgNamespace        = "org-acme"
		orgKubeconfigSecret = "test-kubeconfig"
	)

	ctx := context.Background()

	var err error

	err = createNamespace(ctx, orgNamespace)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	err = createSecret(ctx, orgKubeconfigSecret, orgNamespace)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	config := appConfig{
		appCatalog:      catalog,
		appName:         appName,
		appNamespace:    orgNamespace,
		appVersion:      "1.2.2",
		inCluster:       false,
		targetCluster:   namespace,
		targetNamespace: "default",
	}

	expectedError := "validation error: label `giantswarm.io/cluster` not found"

	err = executeWithApp(ctx, expectedError, config)
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

	err = createNamespace(ctx, orgNamespace)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	config := appConfig{
		appCatalog:      catalog,
		appLabels:       map[string]string{label.AppOperatorVersion: "5.5.0"},
		appName:         appName,
		appNamespace:    orgNamespace,
		appVersion:      "1.2.2",
		inCluster:       true,
		targetNamespace: "kube-system",
	}
	expectedError := "validation error: target namespace kube-system is not allowed for in-cluster apps"

	err = executeWithApp(ctx, expectedError, config)
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

	logger.Debugf(ctx, "creating test resources in %#q namespace", namespace)

	err = createTestResources(ctx)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	logger.Debugf(ctx, "created test resources in %#q namespace", namespace)

	logger.Debugf(ctx, "deleting %#q namespace", namespace)

	err = appTest.K8sClient().CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	logger.Debugf(ctx, "deleted %#q namespace", namespace)

	logger.Debugf(ctx, "waiting for %#q app deletion", appName)

	app := &v1alpha1.App{}

	o := func() error {
		err = appTest.CtrlClient().Get(ctx, types.NamespacedName{Name: appName, Namespace: namespace}, app)
		if apierrors.IsNotFound(err) {
			// fall through
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}
	b := backoff.NewConstant(5*time.Minute, 10*time.Second)
	n := backoff.NewNotifier(logger, ctx)

	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	logger.Debugf(ctx, "waited for %#q app deletion", appName)
}

func createTestResources(ctx context.Context) error {
	var err error

	err = createNamespace(ctx, namespace)
	if err != nil {
		return microerror.Mask(err)
	}

	err = createConfigMap(ctx, configMapName, namespace)
	if err != nil {
		return microerror.Mask(err)
	}

	err = createAppCatalog(ctx, catalog)
	if err != nil {
		return microerror.Mask(err)
	}

	config := appConfig{
		appCatalog:      catalog,
		appLabels:       map[string]string{label.AppOperatorVersion: "5.5.0"},
		appName:         appName,
		appNamespace:    namespace,
		appVersion:      "1.2.2",
		configName:      configMapName,
		inCluster:       true,
		targetNamespace: namespace,
	}

	err = createApp(ctx, config)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
