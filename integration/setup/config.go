//go:build k8srequired
// +build k8srequired

package setup

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/apptest"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"

	"github.com/giantswarm/app-admission-controller/v2/integration/env"
	"github.com/giantswarm/app-admission-controller/v2/integration/helpers"
)

type TestConfig struct {
	AppTest    apptest.Interface
	K8sClients k8sclient.Interface
	Logger     micrologger.Logger
}

func (tc *TestConfig) CreateApp(ctx context.Context, appConfig helpers.AppConfig) error {
	var err error

	app := helpers.GetAppCR(appConfig)

	o := func() error {
		err = tc.AppTest.CtrlClient().Delete(ctx, app)
		if !apierrors.IsNotFound(err) && err != nil {
			return microerror.Mask(err)
		}

		err = tc.AppTest.CtrlClient().Create(ctx, app)
		if !apierrors.IsAlreadyExists(err) && err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	return tc.ensureExecuted(ctx, o)
}

func (tc *TestConfig) CreateCatalog(ctx context.Context, name string) error {
	catalogCR := &v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: v1alpha1.CatalogSpec{
			Description: fmt.Sprintf("%s catalog", name),
			LogoURL:     "/images/repo_icons/giantswarm.png",
			Storage: v1alpha1.CatalogSpecStorage{
				URL:  "",
				Type: "helm",
			},
			Repositories: []v1alpha1.CatalogSpecRepository{
				v1alpha1.CatalogSpecRepository{
					URL:  "",
					Type: "oci",
				},
			},
			Title: name,
		},
	}

	o := func() error {
		err := tc.AppTest.CtrlClient().Create(ctx, catalogCR)
		if !apierrors.IsAlreadyExists(err) && err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	return tc.ensureExecuted(ctx, o)
}

func (tc *TestConfig) CreateCluster(ctx context.Context, name, namespace, releaseVersion string) error {
	if _, err := semver.NewVersion(releaseVersion); err != nil {
		return fmt.Errorf("releaseVersion %q is not a correct semver", releaseVersion)
	}

	clusterCR := &capiv1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				label.ReleaseVersion: releaseVersion,
				label.Cluster:        name,
			},
		},
		Spec: capiv1beta1.ClusterSpec{},
	}

	o := func() error {
		err := tc.AppTest.CtrlClient().Create(ctx, clusterCR)
		if !apierrors.IsAlreadyExists(err) && err != nil {
			return microerror.Mask(err)
		}
		return nil
	}

	return tc.ensureExecuted(ctx, o)
}

func (tc *TestConfig) CreateConfigMap(ctx context.Context, name, namespace string) error {
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
		_, err := tc.AppTest.K8sClient().CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
		if !apierrors.IsAlreadyExists(err) && err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	return tc.ensureExecuted(ctx, o)
}

func (tc *TestConfig) CreateNamespace(ctx context.Context, name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	o := func() error {
		_, err := tc.AppTest.K8sClient().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if !apierrors.IsAlreadyExists(err) && err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	return tc.ensureExecuted(ctx, o)
}

func (tc *TestConfig) CreateSecret(ctx context.Context, name, namespace string) error {
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
		_, err := tc.AppTest.K8sClient().CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if !apierrors.IsAlreadyExists(err) && err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	return tc.ensureExecuted(ctx, o)
}

func (tc *TestConfig) ensureExecuted(ctx context.Context, o func() error) error {
	b := backoff.NewConstant(5*time.Minute, 10*time.Second)
	n := backoff.NewNotifier(tc.Logger, ctx)

	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (tc *TestConfig) GetApp(ctx context.Context, name, namespace string) (*v1alpha1.App, error) {
	var err error

	app := &v1alpha1.App{}

	o := func() error {
		err = tc.AppTest.CtrlClient().Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, app)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	err = tc.ensureExecuted(ctx, o)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return app, nil
}

func NewConfig() (TestConfig, error) {
	var err error

	var logger micrologger.Logger
	{
		c := micrologger.Config{}

		logger, err = micrologger.New(c)
		if err != nil {
			return TestConfig{}, microerror.Mask(err)
		}
	}

	var k8sClients *k8sclient.Clients
	{
		c := k8sclient.ClientsConfig{
			SchemeBuilder: k8sclient.SchemeBuilder{
				v1alpha1.AddToScheme,
				capiv1beta1.AddToScheme,
			},
			Logger: logger,

			KubeConfigPath: env.KubeConfig(),
		}

		k8sClients, err = k8sclient.NewClients(c)
		if err != nil {
			return TestConfig{}, microerror.Mask(err)
		}
	}

	var appTest apptest.Interface
	{
		c := apptest.Config{
			Logger: logger,

			KubeConfigPath: env.KubeConfig(),
		}

		appTest, err = apptest.New(c)
		if err != nil {
			return TestConfig{}, microerror.Mask(err)
		}
	}

	c := TestConfig{
		AppTest:    appTest,
		K8sClients: k8sClients,
		Logger:     logger,
	}

	return c, nil
}
