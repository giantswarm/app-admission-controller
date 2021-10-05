//go:build k8srequired
// +build k8srequired

package validation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
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

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	cm := &corev1.ConfigMap{
		Data: map[string]string{
			"values": "values",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
		},
	}

	// TODO: Remove once apptestctl creates catalog CRs.
	catalogCR := &v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      catalog,
			Namespace: "default",
		},
		Spec: v1alpha1.CatalogSpec{
			Description: "This catalog holds Apps exclusively running on Giant Swarm control planes.",
			LogoURL:     "/images/repo_icons/giantswarm.png",
			Storage: v1alpha1.CatalogSpecStorage{
				URL:  "",
				Type: "helm",
			},
			Title: catalog,
		},
	}

	app := &v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
			Labels: map[string]string{
				label.AppOperatorVersion: "3.0.0",
			},
		},
		Spec: v1alpha1.AppSpec{
			Catalog: catalog,
			Config: v1alpha1.AppSpecConfig{
				ConfigMap: v1alpha1.AppSpecConfigConfigMap{
					Name:      configMapName,
					Namespace: namespace,
				},
			},
			Name:      appName,
			Namespace: namespace,
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				InCluster: true,
			},
			Version: "1.2.2",
		},
	}

	o := func() error {
		_, err = appTest.K8sClient().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// fall through
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		_, err = appTest.K8sClient().CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// fall through
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		err = appTest.CtrlClient().Create(ctx, catalogCR)
		if apierrors.IsAlreadyExists(err) {
			// fall through
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		err = appTest.CtrlClient().Create(ctx, app)
		if apierrors.IsAlreadyExists(err) {
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
		return microerror.Mask(err)
	}

	return nil
}
