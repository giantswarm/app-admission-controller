//go:build k8srequired
// +build k8srequired

package validation

import (
	"context"
	"strings"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/app-admission-controller/v2/integration/helpers"
)

func executeAppTest(ctx context.Context, expectedError string, appConfig helpers.AppConfig) error {
	var err error

	app := helpers.GetAppCR(appConfig)

	config.Logger.Debugf(ctx, "waiting for failed app creation")

	o := func() error {
		err = config.AppTest.CtrlClient().Create(ctx, app)
		if err == nil {
			return microerror.Maskf(executionFailedError, "expected error but got nil")
		}
		if !strings.Contains(err.Error(), expectedError) {
			return microerror.Maskf(executionFailedError, "error == %#v, want %#v ", err.Error(), expectedError)
		}

		return nil
	}
	b := backoff.NewConstant(5*time.Minute, 10*time.Second)
	n := backoff.NewNotifier(config.Logger, ctx)

	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	config.Logger.Debugf(ctx, "waited for failed app creation")

	return nil
}
