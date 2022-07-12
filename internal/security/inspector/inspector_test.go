package inspector

import (
	"context"
	"testing"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
	"k8s.io/apimachinery/pkg/util/errors"
)

func Test_Inspector_hasBlacklistedReference(t *testing.T) {
	tests := []struct {
		name               string
		namespaceBlacklist []string
		app                v1alpha1.App
		errorMatcher       errors.Matcher
	}{
		{
			name:               "test case 1: empty allows all",
			namespaceBlacklist: []string{},
			app: v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "test",
							Namespace: "default",
						},
					},
					UserConfig: v1alpha1.AppSpecUserConfig{
						Secret: v1alpha1.AppSpecUserConfigSecret{
							Name:      "test",
							Namespace: "giantswarm",
						},
					},
				},
			},
		},
		{
			name:               "test case 2: do not allow giantswarm namespace",
			namespaceBlacklist: []string{"giantswarm"},
			app: v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "test",
							Namespace: "default",
						},
					},
					UserConfig: v1alpha1.AppSpecUserConfig{
						Secret: v1alpha1.AppSpecUserConfigSecret{
							Name:      "test",
							Namespace: "giantswarm",
						},
					},
				},
			},
			errorMatcher: IsSecurityViolationError,
		},
		{
			name:               "test case 3: inspect namespace references in extra configs",
			namespaceBlacklist: []string{"giantswarm"},
			app: v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "test",
							Namespace: "default",
						},
					},
					UserConfig: v1alpha1.AppSpecUserConfig{
						Secret: v1alpha1.AppSpecUserConfigSecret{
							Name:      "test",
							Namespace: "test",
						},
					},
					ExtraConfigs: []v1alpha1.AppExtraConfig{
						{
							Kind:      "configMap",
							Name:      "hello",
							Namespace: "world",
						},
						{
							Kind:      "secret",
							Name:      "foo",
							Namespace: "bar",
						},
					},
				},
			},
		},
		{
			name:               "test case 4: catch namespace references in extra configs",
			namespaceBlacklist: []string{"bar"},
			app: v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "test",
							Namespace: "default",
						},
					},
					UserConfig: v1alpha1.AppSpecUserConfig{
						Secret: v1alpha1.AppSpecUserConfigSecret{
							Name:      "test",
							Namespace: "test",
						},
					},
					ExtraConfigs: []v1alpha1.AppExtraConfig{
						{
							Kind:      "configMap",
							Name:      "hello",
							Namespace: "world",
						},
						{
							Kind:      "secret",
							Name:      "foo",
							Namespace: "bar",
						},
					},
				},
			},
			errorMatcher: IsSecurityViolationError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			inspector, err := New(Config{
				Logger:             microloggertest.New(),
				AppBlacklist:       []string{},
				CatalogBlacklist:   []string{},
				GroupWhitelist:     []string{},
				NamespaceBlacklist: tc.namespaceBlacklist,
				UserWhitelist:      []string{},
			})

			if err != nil {
				t.Fatalf("Failed to instantiate inspector: %#v", err)
			}

			err = inspector.hasBlacklistedReference(ctx, tc.app)

			if tc.errorMatcher == nil {
				if err != nil {
					t.Fatalf("Expected not validation errors, but got: %#v", err)
				}
			} else if !tc.errorMatcher(err) {
				t.Fatalf("Did not match expected error, got: %#v", err)
			}
		})
	}
}
