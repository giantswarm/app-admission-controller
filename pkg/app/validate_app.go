package app

import (
	"context"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/api/admission/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-admission-controller/config"
	"github.com/giantswarm/app-admission-controller/pkg/key"
	"github.com/giantswarm/app-admission-controller/pkg/validator"
)

const (
	Name = "app"

	resourceNotFoundTemplate        = "%s %#q in namespace %#q not found"
	namespaceNotFoundReasonTemplate = "namespace is not specified for %s %#q"
)

type Validator struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger
}

func NewValidator(config config.Config) (*Validator, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	validator := &Validator{
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return validator, nil
}

func (v *Validator) Validate(request *v1beta1.AdmissionRequest) (bool, error) {
	ctx := context.Background()

	var app v1alpha1.App

	if _, _, err := validator.Deserializer.Decode(request.Object.Raw, nil, &app); err != nil {
		return false, microerror.Maskf(parsingFailedError, "unable to parse app: %#v", err)
	}

	appAllowed, err := v.ValidateApp(ctx, app)
	if err != nil {
		return false, microerror.Mask(err)
	}

	return appAllowed, nil
}

func (v *Validator) ValidateApp(ctx context.Context, app v1alpha1.App) (bool, error) {
	err := v.validateApp(ctx, app)
	if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

func (m *Validator) Log(keyVals ...interface{}) {
	m.logger.Log(keyVals...)
}

func (v *Validator) Resource() string {
	return Name
}

func (v *Validator) validateApp(ctx context.Context, cr v1alpha1.App) error {
	if key.AppConfigMapName(cr) != "" {
		ns := key.AppConfigMapNamespace(cr)
		if ns == "" {
			return microerror.Maskf(validationError, namespaceNotFoundReasonTemplate, "configmap", key.AppConfigMapName(cr))
		}

		_, err := v.k8sClient.K8sClient().CoreV1().ConfigMaps(ns).Get(ctx, key.AppConfigMapName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return microerror.Maskf(validationError, resourceNotFoundTemplate, "configmap", key.AppConfigMapName(cr), ns)
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if key.AppSecretName(cr) != "" {
		ns := key.AppSecretNamespace(cr)
		if ns == "" {
			return microerror.Maskf(validationError, namespaceNotFoundReasonTemplate, "secret", key.AppSecretName(cr))
		}

		_, err := v.k8sClient.K8sClient().CoreV1().Secrets(ns).Get(ctx, key.AppSecretName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return microerror.Maskf(validationError, resourceNotFoundTemplate, "secret", key.AppSecretName(cr), ns)
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if key.UserConfigMapName(cr) != "" {
		ns := key.UserConfigMapNamespace(cr)
		if ns == "" {
			return microerror.Maskf(validationError, namespaceNotFoundReasonTemplate, "configmap", key.UserConfigMapName(cr))
		}

		_, err := v.k8sClient.K8sClient().CoreV1().ConfigMaps(ns).Get(ctx, key.UserConfigMapName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return microerror.Maskf(validationError, resourceNotFoundTemplate, "configmap", key.UserConfigMapName(cr), ns)
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if key.UserSecretName(cr) != "" {
		ns := key.UserSecretNamespace(cr)
		if ns == "" {
			return microerror.Maskf(validationError, namespaceNotFoundReasonTemplate, "secret", key.UserSecretName(cr))
		}

		_, err := v.k8sClient.K8sClient().CoreV1().Secrets(key.UserSecretNamespace(cr)).Get(ctx, key.UserSecretName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return microerror.Maskf(validationError, resourceNotFoundTemplate, "secret", key.UserSecretName(cr), ns)
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if !key.InCluster(cr) {
		ns := key.KubeConfigSecretNamespace(cr)
		if ns == "" {
			return microerror.Maskf(validationError, namespaceNotFoundReasonTemplate, "kubeconfig secret", key.KubeConfigSecretName(cr))
		}

		_, err := v.k8sClient.K8sClient().CoreV1().Secrets(key.KubeConfigSecretNamespace(cr)).Get(ctx, key.KubeConfigSecretName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return microerror.Maskf(validationError, resourceNotFoundTemplate, "kubeconfig secret", key.KubeConfigSecretName(cr), ns)
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
