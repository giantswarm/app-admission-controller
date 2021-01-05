package mutator

import (
	"context"
	"net/http"
	"reflect"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/micrologger/loggermeta"
	kubewebhookhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-admission-controller/pkg/admissionkit/internal/key"
)

type HttpHandlerConfig struct {
	Chain  []Interface
	Name   string
	Logger micrologger.Logger
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

	var m *mutating.Chain
	{
		var ms []mutating.Mutator
		for _, m := range config.Chain {
			ms = append(ms, newKubewebhookMutator(config.Logger, m))
		}

		m = mutating.NewChain(key.ToKubewebhookLogger(config.Logger), ms...)
	}

	c := mutating.WebhookConfig{
		Name: config.Name,
		Obj:  config.Obj,
	}

	wh, err := mutating.NewWebhook(c, m, nil, nil, key.ToKubewebhookLogger(config.Logger))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	h, err := kubewebhookhttp.HandlerFor(wh)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return h, nil

}

type kubewebhookMutator struct {
	logger     micrologger.Logger
	underlying Interface
}

func newKubewebhookMutator(logger micrologger.Logger, m Interface) *kubewebhookMutator {
	return &kubewebhookMutator{
		logger:     logger,
		underlying: m,
	}
}

func (m *kubewebhookMutator) Mutate(ctx context.Context, obj metav1.Object) (bool, error) {
	// Just in case.
	if key.IsDeleted(obj) {
		return false, nil
	}

	ctx = loggermeta.NewContext(ctx, key.CreateLoggerMeta(obj, m.underlying.Name()+"-mutator"))

	m.logger.Debugf(ctx, "computing mutation")

	req := Request{
		Obj: obj,
	}

	resp, err := m.underlying.Mutate(ctx, req)
	if err != nil {
		// Log and return error because then it's handled by
		// slok/kubewebhook and stack trace wouldn't be printed.
		m.logger.Errorf(ctx, err, "computing mutation failed")
		return false, microerror.Mask(err)
	}

	if resp != nil && resp.MutatedObj != nil {
		// Because underlying kubewebhook library assumes the object
		// will be mutated in place we need to do some reflect magic to
		// copy value of the pointer behind the interface to mutate
		// `obj`.
		originalObj := reflect.ValueOf(obj).Elem()
		mutatedObj := reflect.ValueOf(resp.MutatedObj).Elem()
		originalObj.Set(mutatedObj)
	}

	m.logger.Debugf(ctx, "computed mutation")

	return false, nil
}
