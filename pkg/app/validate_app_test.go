package app

import (
	"strconv"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/app-admission-controller/pkg/unittest"
)

func TestValidateApp(t *testing.T) {
	var err error

	var newLogger micrologger.Logger
	{
		newLogger, err = micrologger.New(micrologger.Config{})
		if err != nil {
			panic(microerror.JSON(err))
		}
	}

	fakeK8sClient := unittest.FakeK8sClient()
	validate := &Validator{
		k8sClient: fakeK8sClient,
		logger:    newLogger,
	}

	testCases := []struct {
		name string

		allowed bool
	}{
		{
			name: "case 0: control plane app",

			allowed: true,
		},
	}
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var err error

			admissionRequest, err := unittest.DefaultAdmissionRequestApp()
			if err != nil {
				t.Fatal(err)
			}

			allowed, _ := validate.Validate(&admissionRequest)
			if allowed != tc.allowed {
				t.Fatalf("expected %v to not to differ from %v", allowed, tc.allowed)
			}
		})
	}
}
