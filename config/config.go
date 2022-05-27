package config

import (
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v6/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	restclient "k8s.io/client-go/rest"
)

const (
	defaultAddress        = ":8443"
	defaultMetricsAddress = ":8080"
)

type Config struct {
	Address        string
	CertFile       string
	KeyFile        string
	MetricsAddress string
	Provider       string

	// Configuration for security validation
	AppBlacklist       []string
	CatalogBlacklist   []string
	GroupWhitelist     []string
	NamespaceBlacklist []string
	UserWhitelist      []string

	Logger    micrologger.Logger
	K8sClient k8sclient.Interface
}

func Parse() (Config, error) {
	var err error
	var config Config

	// Create a new logger that is used by all admitters.
	var newLogger micrologger.Logger
	{
		newLogger, err = micrologger.New(micrologger.Config{})
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
		config.Logger = newLogger
	}

	// Create a new k8sclient that is used by all admitters.
	var k8sClient k8sclient.Interface
	{
		restConfig, err := restclient.InClusterConfig()
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
		c := k8sclient.ClientsConfig{
			SchemeBuilder: k8sclient.SchemeBuilder{
				v1alpha1.AddToScheme,
			},
			Logger: config.Logger,

			RestConfig: restConfig,
		}

		k8sClient, err = k8sclient.NewClients(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
		config.K8sClient = k8sClient
	}

	kingpin.Flag("address", "The address to listen on").Default(defaultAddress).StringVar(&config.Address)
	kingpin.Flag("metrics-address", "The metrics address for Prometheus").Default(defaultMetricsAddress).StringVar(&config.MetricsAddress)
	kingpin.Flag("tls-cert-file", "File containing the certificate for HTTPS").Required().StringVar(&config.CertFile)
	kingpin.Flag("tls-key-file", "File containing the private key for HTTPS").Required().StringVar(&config.KeyFile)
	kingpin.Flag("provider", "Provider of the management cluster. One of aws, azure, kvm").Required().StringVar(&config.Provider)

	kingpin.Flag("whitelist-group", "Whitelisted group").StringsVar(&config.GroupWhitelist)
	kingpin.Flag("whitelist-user", "Whitelisted user").StringsVar(&config.UserWhitelist)
	kingpin.Flag("blacklist-app", "Blacklisted apps").StringsVar(&config.AppBlacklist)
	kingpin.Flag("blacklist-catalog", "Blacklisted catalogs").StringsVar(&config.CatalogBlacklist)
	kingpin.Flag("blacklist-namespace", "Blacklisted namespaces").StringsVar(&config.NamespaceBlacklist)

	kingpin.Parse()

	return config, nil
}
