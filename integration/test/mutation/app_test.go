//go:build k8srequired
// +build k8srequired

package mutation

import (
	"context"
	"testing"

	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-admission-controller/integration/helpers"
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

	config.Logger.Debugf(ctx, "creating test resources in %#q namespace", namespace)

	err = createTestResources(ctx)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	config.Logger.Debugf(ctx, "created test resources in %#q namespace", namespace)

	config.Logger.Debugf(ctx, "creating app")

	// InCluster and DefaultingEnabled set to false.
	// Other fields will be defaulted.
	appConfig := helpers.AppConfig{
		AppCatalog:      catalogName,
		AppLabels:       map[string]string{label.AppOperatorVersion: "3.0.0"},
		AppName:         appName,
		AppNamespace:    namespace,
		AppVersion:      "1.2.2",
		TargetCluster:   namespace,
		TargetNamespace: "giantswarm",
	}

	err = config.CreateApp(ctx, appConfig)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	config.Logger.Debugf(ctx, "created app")

	config.Logger.Debugf(ctx, "checking defaulted values for app")

	app, err := config.GetApp(ctx, appName, namespace)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	if key.KubeConfigSecretNamespace(*app) != namespace {
		t.Fatalf("expected kubeconfig namespace %#q but got %#q", namespace, key.KubeConfigSecretNamespace(*app))
	}
	if key.KubeConfigSecretName(*app) != kubeConfigName {
		t.Fatalf("expected kubeconfig secret name %#q but got %#q", kubeConfigName, key.KubeConfigSecretName(*app))
	}

	config.Logger.Debugf(ctx, "checked defaulted values for app")
}

// TestDefaultKubeConfigOrg checks that the app CR kubeconfig is defaulted to the
// correct settings for the org-namespaced app.
func TestDefaultKubeConfigOrg(t *testing.T) {
	const (
		orgNamespace        = "org-acme"
		orgKubeConfigSecret = "test-kubeconfig"
	)

	ctx := context.Background()

	var err error

	err = config.CreateNamespace(ctx, orgNamespace)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	err = config.CreateSecret(ctx, orgKubeConfigSecret, orgNamespace)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	config.Logger.Debugf(ctx, "creating app")

	// InCluster and DefaultingEnabled set to false.
	// Other fields will be defaulted.
	appConfig := helpers.AppConfig{
		AppCatalog:      catalogName,
		AppLabels:       map[string]string{label.Cluster: "test"},
		AppName:         appName,
		AppNamespace:    orgNamespace,
		AppVersion:      "1.2.2",
		InCluster:       false,
		TargetCluster:   namespace,
		TargetNamespace: "giantswarm",
	}

	err = config.CreateApp(ctx, appConfig)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	config.Logger.Debugf(ctx, "created app")

	config.Logger.Debugf(ctx, "checking defaulted values for app")

	app, err := config.GetApp(ctx, appName, orgNamespace)
	if err != nil {
		t.Fatalf("expected nil but got error %#v", err)
	}

	if key.KubeConfigSecretNamespace(*app) != orgNamespace {
		t.Fatalf("expected kubeconfig namespace %#q but got %#q", orgNamespace, key.KubeConfigSecretNamespace(*app))
	}
	if key.KubeConfigSecretName(*app) != orgKubeConfigSecret {
		t.Fatalf("expected kubeconfig secret name %#q but got %#q", orgKubeConfigSecret, key.KubeConfigSecretName(*app))
	}
	if key.VersionLabel(*app) != "" {
		t.Fatalf("expected empty version label but got %#q", key.VersionLabel(*app))
	}

	config.Logger.Debugf(ctx, "checked defaulted values for app")
}

func createTestResources(ctx context.Context) error {
	var err error

	err = config.CreateNamespace(ctx, namespace)
	if err != nil {
		return microerror.Mask(err)
	}

	err = config.CreateSecret(ctx, kubeConfigName, namespace)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
