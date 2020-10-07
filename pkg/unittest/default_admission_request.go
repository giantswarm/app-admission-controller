package unittest

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/microerror"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

func DefaultAdmissionRequestApp() (v1beta1.AdmissionRequest, error) {
	byt, err := json.Marshal(DefaultApp())
	if err != nil {
		return v1beta1.AdmissionRequest{}, microerror.Mask(err)
	}

	req := v1beta1.AdmissionRequest{
		Kind: metav1.GroupVersionKind{
			Version: "application.giantswarm.io/v1alpha1",
			Kind:    "App",
		},
		Resource: metav1.GroupVersionResource{
			Version:  "application.giantswarm.io/v1alpha1",
			Resource: "apps",
		},
		Operation: v1beta1.Create,
		Object: runtime.RawExtension{
			Raw:    byt,
			Object: nil,
		},
	}
	return req, nil
}
