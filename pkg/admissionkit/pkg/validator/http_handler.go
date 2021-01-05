package validator

import (
	"context"
	"net/http"
	"reflect"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/micrologger/loggermeta"
	kubewebhookhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/webhook/validating"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-admission-controller/pkg/admissionkit/internal/key"
)

type HttpHandlerConfig struct {
	Chain  []Interface
	Logger micrologger.Logger
	Name   string
	Obj    metav1.Object
}

func NewHttpHandler(config HttpHandlerConfig) (http.Handler, error) {
	if len(config.Chain) == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.Chain must not be empty", config)
	}
	if config.Name == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Name must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Obj == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Obj must not be empty", config)
	}
	if reflect.ValueOf(config.Obj).Kind() != reflect.Ptr {
		return nil, microerror.Maskf(invalidConfigError, "%T.Obj must be a pointer, got %#q", config, reflect.ValueOf(config.Obj).Kind())
	}

	var v *validating.Chain
	{
		var vs []validating.Validator
		for _, v := range config.Chain {
			vs = append(vs, newKubewebhookValidator(config.Logger, v))
		}

		v = validating.NewChain(key.ToKubewebhookLogger(config.Logger), vs...)
	}

	c := validating.WebhookConfig{
		Name: config.Name,
		Obj:  config.Obj,
	}

	wh, err := validating.NewWebhook(c, v, nil, nil, key.ToKubewebhookLogger(config.Logger))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	h, err := kubewebhookhttp.HandlerFor(wh)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return h, nil

}

type kubewebhookValidator struct {
	logger     micrologger.Logger
	underlying Interface
}

func newKubewebhookValidator(logger micrologger.Logger, v Interface) *kubewebhookValidator {
	return &kubewebhookValidator{
		logger:     logger,
		underlying: v,
	}
}

func (v *kubewebhookValidator) Validate(ctx context.Context, obj metav1.Object) (stop bool, valid validating.ValidatorResult, err error) {
	// Just in case.
	if key.IsDeleted(obj) {
		return
	}

	ctx = loggermeta.NewContext(ctx, key.CreateLoggerMeta(obj, v.underlying.Name()+"-validator"))

	v.logger.Debugf(ctx, "computing validation")

	req := Request{
		Obj: obj,
	}

	resp, err := v.underlying.Validate(ctx, req)
	if err != nil {
		// Log and return error because then it's handled by
		// slok/kubewebhook and stack trace wouldn't be printed.
		v.logger.Errorf(ctx, err, "computing mutation failed")
		return true, validating.ValidatorResult{}, microerror.Mask(err)
	}

	if resp == nil {
		valid.Valid = true
		return false, valid, nil
	}

	valid = validating.ValidatorResult{
		Valid:   resp.Validation.valid,
		Message: resp.Validation.message,
	}

	if resp.Validation.valid {
		v.logger.Debugf(ctx, "computed validation: accept")
	} else {
		v.logger.Debugf(ctx, "computed validation: reject with message %#q", resp.Validation.message)
	}

	return false, valid, nil
}
