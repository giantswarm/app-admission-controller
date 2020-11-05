// +build k8srequired

package validation

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/giantswarm/apptest"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-admission-controller/integration/env"
)

const (
	prodCatalogName = "control-plane"
	prodCatalogUrl  = "https://giantswarm.github.io/control-plane-catalog"
	testCatalogName = "control-plane-test"
	testCatalogUrl  = "https://giantswarm.github.io/control-plane-test-catalog"
)

var (
	appTest apptest.Interface
	logger  micrologger.Logger
)

// TestMain allows us to have common setup and teardown steps that are run
// once for all the tests https://golang.org/pkg/testing/#hdr-Main.
func TestMain(m *testing.M) {
	var err error

	ctx := context.Background()

	{
		logger, err = micrologger.New(micrologger.Config{})
		if err != nil {
			panic(err.Error())
		}
	}

	{
		appTest, err = apptest.New(apptest.Config{
			KubeConfigPath: env.KubeConfig(),
			Logger:         logger,
		})
		if err != nil {
			panic(err.Error())
		}
	}

	{
		values := `Installation:
  V1:
    Registry:
      Domain: quay.io`
		apps := []apptest.App{
			{
				CatalogName:   prodCatalogName,
				CatalogURL:    prodCatalogUrl,
				Name:          "cert-manager-app",
				Namespace:     metav1.NamespaceSystem,
				Version:       "2.3.1",
				WaitForDeploy: true,
			},
			{
				CatalogName:   testCatalogName,
				CatalogURL:    testCatalogUrl,
				Name:          "app-admission-controller",
				Namespace:     "giantswarm",
				SHA:           env.CircleSHA(),
				ValuesYAML:    values,
				WaitForDeploy: true,
			},
		}
		err = appTest.InstallApps(ctx, apps)
		if err != nil {
			logger.LogCtx(ctx, "level", "error", "message", "install apps failed", "stack", fmt.Sprintf("%#v\n", err))
			os.Exit(-1)
		}
	}

	os.Exit(m.Run())
}
