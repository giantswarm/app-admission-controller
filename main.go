package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/devctl/pkg/project"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/giantswarm/app-admission-controller/config"
	"github.com/giantswarm/app-admission-controller/pkg/admissionkit/pkg/mutator"
	"github.com/giantswarm/app-admission-controller/pkg/admissionkit/pkg/validator"
	"github.com/giantswarm/app-admission-controller/pkg/app"
)

func main() {
	err := mainWithError()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", microerror.Pretty(err, true))
		os.Exit(2)
	}
}

func mainWithError() error {
	ctx := context.Background()

	cfg, err := config.Parse()
	if err != nil {
		panic(microerror.JSON(err))
	}

	var logger micrologger.Logger
	{
		logger, err = micrologger.New(micrologger.Config{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var appMutator *app.Mutator
	{
		c := app.MutatorConfig{
			K8sClient: cfg.K8sClient,
			Logger:    logger,
		}
		appMutator, err = app.NewMutator(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var appValidator *app.Validator
	{
		c := app.ValidatorConfig{
			K8sClient: cfg.K8sClient,
			Logger:    logger,
		}
		appValidator, err = app.NewValidator(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var mutatorHandler http.Handler
	{
		c := mutator.HttpHandlerConfig{
			Chain: []mutator.Interface{
				appMutator,
			},
			Name:   project.Name() + "-mutator",
			Logger: logger,
			Obj:    new(v1alpha1.App),
		}

		mutatorHandler, err = mutator.NewHttpHandler(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var validatorHandler http.Handler
	{
		c := validator.HttpHandlerConfig{
			Chain: []validator.Interface{
				appValidator,
			},
			Name:   project.Name() + "-validator",
			Logger: logger,
			Obj:    new(v1alpha1.App),
		}

		validatorHandler, err = validator.NewHttpHandler(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Here we register our endpoints.
	handler := http.NewServeMux()
	handler.Handle("/mutate/app", mutatorHandler)
	handler.Handle("/validate/app", validatorHandler)

	handler.HandleFunc("/healthz", healthCheck)

	metrics := http.NewServeMux()
	metrics.Handle("/metrics", promhttp.Handler())

	logger.Debugf(ctx, "listening on port %s", cfg.Address)

	go serveMetrics(cfg, metrics)
	serveTLS(cfg, handler)

	return nil
}

func healthCheck(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusOK)
	_, err := writer.Write([]byte("ok"))
	if err != nil {
		panic(microerror.JSON(err))
	}
}

func serveTLS(config config.Config, handler http.Handler) {
	server := &http.Server{
		Addr:    config.Address,
		Handler: handler,
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)
	go func() {
		<-sig
		err := server.Shutdown(context.Background())
		if err != nil {
			panic(microerror.JSON(err))
		}
	}()

	err := server.ListenAndServeTLS(config.CertFile, config.KeyFile)
	if err != nil {
		if err != http.ErrServerClosed {
			panic(microerror.JSON(err))
		}
	}
}

func serveMetrics(config config.Config, handler http.Handler) {
	server := &http.Server{
		Addr:    config.MetricsAddress,
		Handler: handler,
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)
	go func() {
		<-sig
		err := server.Shutdown(context.Background())
		if err != nil {
			panic(microerror.JSON(err))
		}
	}()

	err := server.ListenAndServe()
	if err != nil {
		if err != http.ErrServerClosed {
			panic(microerror.JSON(err))
		}
	}
}
