// +build k8srequired

package validation

import (
	"context"
	"strings"
	"testing"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFailWhenCatalogNotFound(t *testing.T) {
	ctx := context.Background()
	var err error

	expectedError := "validation error: catalog `missing` not found"

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
	err = appTest.CtrlClient().Create(ctx, app)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if !strings.Contains(err.Error(), expectedError) {
		t.Fatalf("error == %#v, want %#v ", err.Error(), expectedError)
	}
}
