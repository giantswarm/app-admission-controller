// +build k8srequired

package mutation

import (
	"context"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	appName        = "kiam-app"
	catalogName    = "control-plane-catalog"
	kubeConfigName = "mutation-test-kubeconfig"
	namespace      = "mutation-test"
)

// TestDefaultKubeConfig checks that the app CR kubeconfig is defaulted to the
// correct settings.
func TestDefaultKubeConfig(t *testing.T) {
	ctx := context.Background()

	var err error

	logger.Debugf(ctx, "creating test resources in %#q namespace", namespace)

	err = createTestResources(ctx)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	logger.Debugf(ctx, "created test resources in %#q namespace", namespace)

	logger.Debugf(ctx, "creating app")

	app := v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
			Labels: map[string]string{
				label.AppOperatorVersion: "3.0.0",
			},
		},
		Spec: v1alpha1.AppSpec{
			Catalog:   catalogName,
			Name:      appName,
			Namespace: "giantswarm",
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				// Only set InCluster to false. Other fields will be defaulted.
				InCluster: false,
			},
			Version: "1.2.2",
		},
	}

	o := func() error {
		// First ensure app CR is deleted.
		err = appTest.CtrlClient().Delete(ctx, &app)
		if apierrors.IsNotFound(err) {
			// Fall through.
		} else if err != nil {
			return microerror.Mask(err)
		}

		err = appTest.CtrlClient().Create(ctx, &app)
		if apierrors.IsAlreadyExists(err) {
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

	logger.Debugf(ctx, "created app")

	logger.Debugf(ctx, "checking defaulted values for app")

	if key.KubeConfigSecretNamespace(app) != namespace {
		t.Fatalf("expected kubeconfig namespace %#q but got %#q", namespace, key.KubeConfigSecretNamespace(app))
	}
	if key.KubeConfigSecretName(app) != kubeConfigName {
		t.Fatalf("expected kubeconfig secret name %#q but got %#q", kubeConfigName, key.KubeConfigSecretName(app))
	}

	logger.Debugf(ctx, "checked defaulted values for app")
}

func createTestResources(ctx context.Context) error {
	var err error

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	kubeConfig := &corev1.Secret{
		Data: map[string][]byte{
			"kubeconfig": []byte("kubeconfig"),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeConfigName,
			Namespace: namespace,
		},
	}

	o := func() error {
		_, err = appTest.K8sClient().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Fall through.
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		_, err = appTest.K8sClient().CoreV1().Secrets(namespace).Create(ctx, kubeConfig, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// Fall through.
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
