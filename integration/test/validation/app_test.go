// +build k8srequired

package validation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestFailWhenCatalogNotFound tests that the app CR is rejected if the
// referenced appcatalog CR does not exist.
func TestFailWhenCatalogNotFound(t *testing.T) {
	ctx := context.Background()

	var err error

	app := &v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dex-app-unique",
			Namespace: "giantswarm",
			Labels: map[string]string{
				label.AppOperatorVersion: "0.0.0",
			},
		},
		Spec: v1alpha1.AppSpec{
			Catalog:   "missing",
			Name:      "dex-app",
			Namespace: "giantswarm",
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				InCluster: true,
			},
			Version: "1.2.2",
		},
	}
	expectedError := "validation error: catalog `missing` not found"

	logger.LogCtx(ctx, "level", "debug", "message", "waiting for failed app creation")

	o := func() error {
		err = appTest.CtrlClient().Create(ctx, app)
		if err == nil {
			microerror.Maskf(executionFailedError, "expected error but got nil")
		}
		if !strings.Contains(err.Error(), expectedError) {
			return microerror.Maskf(executionFailedError, "error == %#v, want %#v ", err.Error(), expectedError)
		}

		return nil
	}
	b := backoff.NewConstant(5*time.Minute, 30*time.Second)
	n := backoff.NewNotifier(logger, ctx)

	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	logger.LogCtx(ctx, "level", "debug", "message", "waited for failed app creation")
}

// TestSkipValidationOnDelete tests that the validation is skipped when the app
// CR is deleted. Both an app CR and configmap are created and the configmap is
// deleted first.
func TestSkipValidationOnDelete(t *testing.T) {
	ctx := context.Background()

	var err error

	cm := &corev1.ConfigMap{
		Data: map[string]string{
			"values": "values",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dex-config",
			Namespace: "giantswarm",
		},
	}

	app := &v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dex-app-unique",
			Namespace: "giantswarm",
			Labels: map[string]string{
				label.AppOperatorVersion: "0.0.0",
			},
		},
		Spec: v1alpha1.AppSpec{
			Catalog: "control-plane-catalog",
			Config: v1alpha1.AppSpecConfig{
				ConfigMap: v1alpha1.AppSpecConfigConfigMap{
					Name:      "dex-config",
					Namespace: "giantswarm",
				},
			},
			Name:      "dex-app",
			Namespace: "giantswarm",
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				InCluster: true,
			},
			Version: "1.2.2",
		},
	}

	logger.LogCtx(ctx, "level", "debug", "message", "creating app and configmap")

	o := func() error {
		_, err = appTest.K8sClient().CoreV1().ConfigMaps("giantswarm").Create(ctx, cm, metav1.CreateOptions{})
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
	b := backoff.NewConstant(5*time.Minute, 30*time.Second)
	n := backoff.NewNotifier(logger, ctx)

	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	logger.LogCtx(ctx, "level", "debug", "message", "created app and configmap")

	logger.LogCtx(ctx, "level", "debug", "message", "deleting configmap")

	err = appTest.K8sClient().CoreV1().ConfigMaps("giantswarm").Delete(ctx, "dex-config", metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	logger.LogCtx(ctx, "level", "debug", "message", "deleted configmap")

	logger.LogCtx(ctx, "level", "debug", "message", "deleting app")

	err = appTest.CtrlClient().Delete(ctx, app)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	logger.LogCtx(ctx, "level", "debug", "message", "deleted app")
}
