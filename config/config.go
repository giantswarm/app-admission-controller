package config

import (
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"gopkg.in/yaml.v3"
	restclient "k8s.io/client-go/rest"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
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

	// Configuration for PSP removal
	PSPConfigFile string
	PSPPatches    []ConfigPatch

	Logger    micrologger.Logger
	K8sClient k8sclient.Interface
}

type ConfigPatch struct {
	// AppName is used to match against App CR's .ObjectMeta.Name
	AppName string `yaml:"app_name"`
	// ConfigMapSuffix is a suffix of patch ConfigMap's name
	ConfigMapSuffix string `yaml:"configmap_suffix"`
	// Patch contains Helm values to use as App's extraConfig
	Values string `yaml:"values"`
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
				capiv1beta1.AddToScheme,
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
	kingpin.Flag("psp-config-file", "File containing PSP patch configuration").StringVar(&config.PSPConfigFile)

	kingpin.Parse()

	config.PSPPatches = []ConfigPatch{}

	if config.PSPConfigFile != "" {
		data, err := os.ReadFile(config.PSPConfigFile)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}

		patches := []ConfigPatch{}
		err = yaml.Unmarshal(data, &patches)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
		config.PSPPatches = patches
	}

	return config, nil
}
