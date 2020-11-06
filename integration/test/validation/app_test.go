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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
