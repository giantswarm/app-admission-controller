// +build k8srequired

package mutation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/app/v3/pkg/key"
	"github.com/giantswarm/backoff"
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

func TestDefaultKubeConfig(t *testing.T) {
	ctx := context.Background()

	var err error

	logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating test resources in %#q namespace", namespace))

	err = createTestResources(ctx)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created test resources in %#q namespace", namespace))

	app := v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
			Labels: map[string]string{
				label.AppOperatorVersion: "2.6.0",
			},
		},
		Spec: v1alpha1.AppSpec{
			Catalog:   catalogName,
			Name:      appName,
			Namespace: "giantswarm",
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				InCluster: false,
			},
			Version: "1.2.2",
		},
	}

	logger.LogCtx(ctx, "level", "debug", "message", "creating app")

	o := func() error {
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

	logger.LogCtx(ctx, "level", "debug", "message", "created app")

	logger.LogCtx(ctx, "level", "debug", "message", "checking defaulted values for app")

	if key.KubeConfigSecretNamespace(app) != namespace {
		t.Fatalf("expected kubeconfig namespace %#q but got %#q", namespace, key.KubeConfigSecretNamespace(app))
	}
	if key.KubeConfigSecretName(app) != kubeConfigName {
		t.Fatalf("expected kubeconfig secret name %#q but got %#q", kubeConfigName, key.KubeConfigSecretName(app))
	}

	logger.LogCtx(ctx, "level", "debug", "message", "checked defaulted values for app")
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
			// fall through
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		_, err = appTest.K8sClient().CoreV1().Secrets(namespace).Create(ctx, kubeConfig, metav1.CreateOptions{})
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
