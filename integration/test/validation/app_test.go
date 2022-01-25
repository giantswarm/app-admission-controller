//go:build k8srequired
// +build k8srequired

package validation

import (
	"context"
	"strings"
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

	app := &v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: "giantswarm",
			Labels: map[string]string{
				label.AppOperatorVersion: "3.0.0",
			},
		},
		Spec: v1alpha1.AppSpec{
			Catalog:   "missing",
			Name:      appName,
			Namespace: "giantswarm",
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				InCluster: true,
			},
			Version: "1.2.2",
		},
	}
	expectedError := "validation error: catalog `missing` not found"

	logger.Debugf(ctx, "waiting for failed app creation")

	o := func() error {
		err = appTest.CtrlClient().Create(ctx, app)
		if err == nil {
			return microerror.Maskf(executionFailedError, "expected error but got nil")
		}
		if !strings.Contains(err.Error(), expectedError) {
			return microerror.Maskf(executionFailedError, "error == %#v, want %#v ", err.Error(), expectedError)
		}

		return nil
	}
	b := backoff.NewConstant(5*time.Minute, 10*time.Second)
	n := backoff.NewNotifier(logger, ctx)

	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	logger.Debugf(ctx, "waited for failed app creation")
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

	app := &v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: "org-acme",
		},
		Spec: v1alpha1.AppSpec{
			Catalog:   catalog,
			Name:      appName,
			Namespace: "default",
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				Context: v1alpha1.AppSpecKubeConfigContext{
					Name: orgKubeconfigSecret,
				},
				InCluster: false,
				Secret: v1alpha1.AppSpecKubeConfigSecret{
					Name:      orgKubeconfigSecret,
					Namespace: orgNamespace,
				},
			},
			Version: "1.2.2",
		},
	}
	expectedError := "validation error: label `giantswarm.io/cluster` not found"

	logger.Debugf(ctx, "waiting for failed app creation")

	o := func() error {
		err = appTest.CtrlClient().Create(ctx, app)
		if err == nil {
			return microerror.Maskf(executionFailedError, "expected error but got nil")
		}
		if !strings.Contains(err.Error(), expectedError) {
			return microerror.Maskf(executionFailedError, "error == %#v, want %#v ", err.Error(), expectedError)
		}

		return nil
	}
	b := backoff.NewConstant(5*time.Minute, 10*time.Second)
	n := backoff.NewNotifier(logger, ctx)

	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	logger.Debugf(ctx, "waited for failed app creation")
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

	app := &v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: orgNamespace,
			Labels: map[string]string{
				label.AppOperatorVersion: "5.5.0",
			},
		},
		Spec: v1alpha1.AppSpec{
			Catalog:   catalog,
			Name:      appName,
			Namespace: "kube-system",
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				InCluster: true,
			},
			Version: "1.2.2",
		},
	}
	expectedError := "validation error: target namespace kube-system is not allowed for in-cluster apps"

	logger.Debugf(ctx, "waiting for failed app creation")

	o := func() error {
		err = appTest.CtrlClient().Create(ctx, app)
		if err == nil {
			return microerror.Maskf(executionFailedError, "expected error but got nil")
		}
		if !strings.Contains(err.Error(), expectedError) {
			return microerror.Maskf(executionFailedError, "error == %#v, want %#v ", err.Error(), expectedError)
		}

		return nil
	}
	b := backoff.NewConstant(5*time.Minute, 10*time.Second)
	n := backoff.NewNotifier(logger, ctx)

	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	logger.Debugf(ctx, "waited for failed app creation")
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
