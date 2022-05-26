package inspector

import (
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type empty struct{}

type Config struct {
	Logger micrologger.Logger

	AppBlacklist       []string
	CatalogBlacklist   []string
	GroupWhitelist     []string
	NamespaceBlacklist []string
	UserWhitelist      []string
}

type Inspector struct {
	logger micrologger.Logger

	appBlacklist              map[string]empty
	catalogBlacklist          map[string]empty
	dynamicNamespaceBlacklist []string
	fixedNamespaceBlacklist   map[string]empty
	groupWhitelist            map[string]empty
	userWhitelist             []string
}

func New(config Config) (*Inspector, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	inspector := Inspector{
		logger: config.Logger,

		appBlacklist:              make(map[string]empty),
		catalogBlacklist:          make(map[string]empty),
		dynamicNamespaceBlacklist: make([]string, 0),
		fixedNamespaceBlacklist:   make(map[string]empty),
		groupWhitelist:            make(map[string]empty),
		userWhitelist:             config.UserWhitelist,
	}

	for _, i := range config.NamespaceBlacklist {
		if strings.HasPrefix(i, "-") || strings.HasSuffix(i, "-") {
			inspector.dynamicNamespaceBlacklist = append(inspector.dynamicNamespaceBlacklist, i)
		} else {
			inspector.fixedNamespaceBlacklist[i] = empty{}
		}
	}

	for _, i := range config.GroupWhitelist {
		inspector.groupWhitelist[i] = empty{}
	}

	for _, i := range config.AppBlacklist {
		inspector.appBlacklist[i] = empty{}
	}

	for _, i := range config.CatalogBlacklist {
		inspector.catalogBlacklist[i] = empty{}
	}

	return &inspector, nil
}
