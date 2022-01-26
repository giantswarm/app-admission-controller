//go:build k8srequired
// +build k8srequired

package validation

import (
	"context"
	"strings"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
)

type appConfig struct {
	appCatalog      string
	appLabels       map[string]string
	appName         string
	appNamespace    string
	appVersion      string
	configName      string
	inCluster       bool
	targetCluster   string
	targetNamespace string
}

func getAppCR(config appConfig) *v1alpha1.App {
	app := &v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.appName,
			Namespace: config.appNamespace,
			Labels:    config.appLabels,
		},
		Spec: v1alpha1.AppSpec{
			Catalog:   config.appCatalog,
			Name:      config.appName,
			Namespace: config.targetNamespace,
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				InCluster: config.inCluster,
			},
			Version: config.appVersion,
		},
	}

	if config.configName != "" {
		app.Spec.Config = v1alpha1.AppSpecConfig{
			ConfigMap: v1alpha1.AppSpecConfigConfigMap{
				Name:      config.configName,
				Namespace: config.appNamespace,
			},
		}
	}

	if !config.inCluster {
		app.Spec.KubeConfig.Context = v1alpha1.AppSpecKubeConfigContext{
			Name: config.targetCluster,
		}

		app.Spec.KubeConfig.Secret = v1alpha1.AppSpecKubeConfigSecret{
			Name:      config.targetCluster + "-kubeconfig",
			Namespace: config.appNamespace,
		}
	}

	return app
}

func createNamespace(ctx context.Context, name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	o := func() error {
		_, err := appTest.K8sClient().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if !apierrors.IsAlreadyExists(err) && err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	return ensureCreated(ctx, o)
}

func createSecret(ctx context.Context, name, namespace string) error {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"kubeconfig": []byte("cluster: yaml\n"),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	o := func() error {
		_, err := appTest.K8sClient().CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if !apierrors.IsAlreadyExists(err) && err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	return ensureCreated(ctx, o)
}

func createConfigMap(ctx context.Context, name, namespace string) error {
	cm := &corev1.ConfigMap{
		Data: map[string]string{
			"values": "values",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	o := func() error {
		_, err := appTest.K8sClient().CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
		if !apierrors.IsAlreadyExists(err) && err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	return ensureCreated(ctx, o)
}

func createAppCatalog(ctx context.Context, name string) error {
	// TODO: Remove once apptestctl creates catalog CRs.
	catalogCR := &v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: v1alpha1.CatalogSpec{
			Description: "This catalog holds Apps exclusively running on Giant Swarm control planes.",
			LogoURL:     "/images/repo_icons/giantswarm.png",
			Storage: v1alpha1.CatalogSpecStorage{
				URL:  "",
				Type: "helm",
			},
			Title: name,
		},
	}

	o := func() error {
		err := appTest.CtrlClient().Create(ctx, catalogCR)
		if !apierrors.IsAlreadyExists(err) && err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	return ensureCreated(ctx, o)
}

func createApp(ctx context.Context, config appConfig) error {
	app := getAppCR(config)

	o := func() error {
		err := appTest.CtrlClient().Create(ctx, app)
		if !apierrors.IsAlreadyExists(err) && err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	return ensureCreated(ctx, o)
}

func ensureCreated(ctx context.Context, o func() error) error {
	b := backoff.NewConstant(5*time.Minute, 10*time.Second)
	n := backoff.NewNotifier(logger, ctx)

	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func executeWithApp(ctx context.Context, expectedError string, config appConfig) error {
	var err error

	app := getAppCR(config)

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
		return microerror.Mask(err)
	}

	logger.Debugf(ctx, "waited for failed app creation")

	return nil
}
