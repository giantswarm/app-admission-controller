//go:build k8srequired
// +build k8srequired

package validation

import (
	"testing"

	"github.com/giantswarm/app-admission-controller/v2/integration/setup"
)

var (
	config setup.TestConfig
)

func init() {
	var err error

	{
		config, err = setup.NewConfig()
		if err != nil {
			panic(err.Error())
		}
	}
}

func TestMain(m *testing.M) {
	setup.Setup(m, config)
}
