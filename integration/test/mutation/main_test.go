//go:build k8srequired
// +build k8srequired

package mutation

import (
	"testing"

	"github.com/giantswarm/app-admission-controller/integration/setup"
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
