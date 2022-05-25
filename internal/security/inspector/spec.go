package inspector

import (
	"strings"
)

type empty struct{}

type Config struct {
	AppBlacklist       []string
	CatalogBlacklist   []string
	GroupWhitelist     []string
	NamespaceBlacklist []string
	UserWhitelist      []string
}

type Inspector struct {
	appBlacklist              map[string]empty
	catalogBlacklist          map[string]empty
	dynamicNamespaceBlacklist []string
	fixedNamespaceBlacklist   map[string]empty
	groupWhitelist            map[string]empty
	userWhitelist             []string
}

func New(config Config) *Inspector {
	inspector := Inspector{
		appBlacklist:              make(map[string]empty, 0),
		catalogBlacklist:          make(map[string]empty, 0),
		dynamicNamespaceBlacklist: make([]string, 0),
		fixedNamespaceBlacklist:   make(map[string]empty, 0),
		groupWhitelist:            make(map[string]empty, 0),
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

	return &inspector
}
