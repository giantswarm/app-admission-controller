package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/giantswarm/microerror"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/giantswarm/app-admission-controller/config"
	"github.com/giantswarm/app-admission-controller/pkg/app"
	"github.com/giantswarm/app-admission-controller/pkg/mutator"
	"github.com/giantswarm/app-admission-controller/pkg/validator"
)

func main() {
	config, err := config.Parse()
	if err != nil {
		panic(microerror.JSON(err))
	}

	appMutator, err := app.NewMutator(config)
	if err != nil {
		panic(microerror.JSON(err))
	}

	appValidator, err := app.NewValidator(config)
	if err != nil {
		panic(microerror.JSON(err))
	}

	// Here we register our endpoints.
	handler := http.NewServeMux()
	handler.Handle("/mutate/app", mutator.Handler(appMutator))
	handler.Handle("/validate/app", validator.Handler(appValidator))

	handler.HandleFunc("/healthz", healthCheck)

	metrics := http.NewServeMux()
	metrics.Handle("/metrics", promhttp.Handler())

	go serveMetrics(config, metrics)
	serveTLS(config, handler)
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
