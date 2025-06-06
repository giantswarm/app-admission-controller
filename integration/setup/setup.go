//go:build k8srequired
// +build k8srequired

package setup

import (
	"context"
	"os"
	"testing"

	"github.com/giantswarm/apptest"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-admission-controller/v2/integration/env"
	"github.com/giantswarm/app-admission-controller/v2/integration/templates"
)

func Setup(m *testing.M, config TestConfig) {
	var err error

	ctx := context.Background()

	err = config.CreateNamespace(ctx, "xyz12")
	if err != nil {
		config.Logger.Errorf(ctx, err, "create namespace failed")
		os.Exit(2)
	}

	err = config.CreateCluster(ctx, "xyz12", "default", "v19.2.0")
	if err != nil {
		config.Logger.Errorf(ctx, err, "create cluster failed")
		os.Exit(2)
	}

	err = installResources(ctx, config)
	if err != nil {
		config.Logger.Errorf(ctx, err, "install apps failed")
		os.Exit(2)
	}

	os.Exit(m.Run())
}

func installResources(ctx context.Context, testConfig TestConfig) error {
	apps := []apptest.App{
		{
			CatalogName:   "control-plane-catalog",
			Name:          "cert-manager-app",
			Namespace:     metav1.NamespaceSystem,
			Version:       "3.8.1",
			ValuesYAML:    templates.CertManagerValues,
			WaitForDeploy: true,
		},
		{
			CatalogName:   "control-plane-test-catalog",
			Name:          "app-admission-controller",
			Namespace:     "giantswarm",
			SHA:           env.CircleSHA(),
			ValuesYAML:    templates.AppAdmissionControllerValues,
			WaitForDeploy: true,
		},
	}
	err := testConfig.AppTest.InstallApps(ctx, apps)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
