package key

import (
	"context"
	"strconv"
	"sync/atomic"

	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/micrologger/loggermeta"
	"github.com/slok/kubewebhook/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var requestIDSeq uint64

func IsDeleted(obj metav1.Object) bool {
	if obj.GetDeletionTimestamp() == nil {
		return false
	}
	if obj.GetDeletionTimestamp().IsZero() {
		return false
	}
	return true
}

func CreateLoggerMeta(obj metav1.Object, reviewer string) *loggermeta.LoggerMeta {
	requestID := atomic.AddUint64(&requestIDSeq, 1)

	// selfLink before creation so construct something that's always
	// available instead.
	object := obj.GetNamespace()
	if object != "" {
		object += "/"
	}
	object += obj.GetName()

	// Resource version is not set before creation.
	resourceVersion := obj.GetResourceVersion()
	if resourceVersion == "" {
		resourceVersion = "n/a"
	}

	m := loggermeta.New()
	m.KeyVals["resource"] = reviewer
	m.KeyVals["object"] = object
	m.KeyVals["request"] = strconv.FormatUint(requestID, 10)
	m.KeyVals["version"] = resourceVersion

	return m
}

type kubewebhookLogger struct {
	ctx        context.Context
	underlying micrologger.Logger
}

func newKubewebhookLogger(logger micrologger.Logger) *kubewebhookLogger {
	return &kubewebhookLogger{
		underlying: logger.WithIncreasedCallerDepth(),
	}
}

func (l *kubewebhookLogger) Debugf(format string, args ...interface{}) {
	l.underlying.Debugf(context.Background(), format, args...)
}
func (l *kubewebhookLogger) Errorf(format string, args ...interface{}) {
	l.underlying.Errorf(context.Background(), nil, format, args...)
}
func (l *kubewebhookLogger) Infof(format string, args ...interface{}) {
	l.underlying.Debugf(context.Background(), format, args...)
}
func (l *kubewebhookLogger) Warningf(format string, args ...interface{}) {
	l.underlying.Debugf(context.Background(), format, args...)
}

func ToKubewebhookLogger(logger micrologger.Logger) log.Logger {
	return newKubewebhookLogger(logger)
}
