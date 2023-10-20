package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dyson/certman"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/giantswarm/app-admission-controller/config"
	"github.com/giantswarm/app-admission-controller/pkg/app"
	"github.com/giantswarm/app-admission-controller/pkg/mutator"
	"github.com/giantswarm/app-admission-controller/pkg/validator"

	"github.com/giantswarm/app-admission-controller/internal/recorder"
	secins "github.com/giantswarm/app-admission-controller/internal/security/inspector"
)

func main() {
	err := mainWithError()
	if err != nil {
		panic(fmt.Sprintf("%#v\n", err))
	}
}

func mainWithError() error {
	ctx := context.Background()

	cfg, err := config.Parse()
	if err != nil {
		panic(microerror.JSON(err))
	}

	var newLogger micrologger.Logger
	{
		newLogger, err = micrologger.New(micrologger.Config{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var event recorder.Interface
	{
		c := recorder.Config{
			K8sClient: cfg.K8sClient,

			Component: "app-admission-controller",
		}

		event = recorder.New(c)
	}

	var appMutator *app.Mutator
	{
		c := app.MutatorConfig{
			K8sClient:     cfg.K8sClient,
			Logger:        newLogger,
			Provider:      cfg.Provider,
			ConfigPatches: cfg.PSPPatches,
		}
		appMutator, err = app.NewMutator(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var inspector *secins.Inspector
	{

		c := secins.Config{
			Logger: newLogger,

			NamespaceBlacklist: cfg.NamespaceBlacklist,
			GroupWhitelist:     cfg.GroupWhitelist,
			UserWhitelist:      cfg.UserWhitelist,
			AppBlacklist:       cfg.AppBlacklist,
			CatalogBlacklist:   cfg.CatalogBlacklist,
		}

		inspector, err = secins.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var appValidator *app.Validator
	{
		c := app.ValidatorConfig{
			Event:     event,
			K8sClient: cfg.K8sClient,
			Logger:    newLogger,

			Provider:  cfg.Provider,
			Inspector: inspector,
		}
		appValidator, err = app.NewValidator(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Here we register our endpoints.
	handler := http.NewServeMux()
	handler.Handle("/mutate/app", mutator.Handler(appMutator))
	handler.Handle("/validate/app", validator.Handler(appValidator))

	handler.HandleFunc("/healthz", healthCheck)

	metrics := http.NewServeMux()
	metrics.Handle("/metrics", promhttp.Handler())

	newLogger.Debugf(ctx, "listening on port %s", cfg.Address)

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
	cm, err := certman.New(config.CertFile, config.KeyFile)
	if err != nil {
		panic(microerror.JSON(err))
	}
	if err := cm.Watch(); err != nil {
		panic(microerror.JSON(err))
	}

	server := &http.Server{
		Addr:    config.Address,
		Handler: handler,
		TLSConfig: &tls.Config{
			GetCertificate: cm.GetCertificate,
			MinVersion:     tls.VersionTLS12,
		},
		ReadHeaderTimeout: 60 * time.Second,
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

	err = server.ListenAndServeTLS("", "")
	if err != nil {
		if err != http.ErrServerClosed {
			panic(microerror.JSON(err))
		}
	}
}

func serveMetrics(config config.Config, handler http.Handler) {
	server := &http.Server{
		Addr:              config.MetricsAddress,
		Handler:           handler,
		ReadHeaderTimeout: 60 * time.Second,
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
