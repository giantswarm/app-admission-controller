package mutator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/giantswarm/microerror"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-admission-controller/pkg/metrics"
)

type Mutator interface {
	Debugf(ctx context.Context, format string, params ...interface{})
	Errorf(ctx context.Context, err error, format string, params ...interface{})
	Mutate(review *admissionv1.AdmissionRequest) ([]PatchOperation, error)
	Resource() string
}

var (
	scheme        = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(scheme)
	Deserializer  = codecs.UniversalDeserializer()
	InternalError = errors.New("internal admission controller error")
)

func Handler(mutator Mutator) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ctx := context.Background()
		start := time.Now()
		defer metrics.DurationRequests.WithLabelValues("mutating", mutator.Resource()).Observe(float64(time.Since(start)) / float64(time.Second))

		metrics.TotalRequests.WithLabelValues("mutating", mutator.Resource()).Inc()
		if request.Header.Get("Content-Type") != "application/json" {
			mutator.Errorf(ctx, nil, "invalid content-type: %s", request.Header.Get("Content-Type"))
			metrics.InvalidRequests.WithLabelValues("mutating", mutator.Resource()).Inc()
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		data, err := io.ReadAll(request.Body)
		if err != nil {
			mutator.Errorf(ctx, err, "unable to read request")
			metrics.InternalError.WithLabelValues("mutating", mutator.Resource()).Inc()
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		review := admissionv1.AdmissionReview{}
		if _, _, err := Deserializer.Decode(data, nil, &review); err != nil {
			mutator.Errorf(ctx, err, "unable to parse admission review request")
			metrics.InvalidRequests.WithLabelValues("mutating", mutator.Resource()).Inc()
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		resourceName := fmt.Sprintf("%s %s/%s", review.Request.Kind, review.Request.Namespace, extractName(review.Request))

		patch, err := mutator.Mutate(review.Request)
		if err != nil {
			writeResponse(ctx, mutator, writer, errorResponse(review.Request.UID, microerror.Mask(err)))
			metrics.RejectedRequests.WithLabelValues("mutating", mutator.Resource()).Inc()
			return
		}

		patchData, err := json.Marshal(patch)
		if err != nil {
			mutator.Errorf(ctx, err, "unable to serialize patch for %s", resourceName)
			writeResponse(ctx, mutator, writer, errorResponse(review.Request.UID, InternalError))
			metrics.RejectedRequests.WithLabelValues("mutating", mutator.Resource()).Inc()
			return
		}

		mutator.Debugf(ctx, "admitted %s (with %d patches)", resourceName, len(patch))
		metrics.SuccessfulRequests.WithLabelValues("mutating", mutator.Resource()).Inc()

		pt := admissionv1.PatchTypeJSONPatch
		writeResponse(ctx, mutator, writer, &admissionv1.AdmissionResponse{
			Allowed:   true,
			UID:       review.Request.UID,
			Patch:     patchData,
			PatchType: &pt,
		})
	}
}

func extractName(request *admissionv1.AdmissionRequest) string {
	if request.Name != "" {
		return request.Name
	}

	obj := metav1beta1.PartialObjectMetadata{}
	if _, _, err := Deserializer.Decode(request.Object.Raw, nil, &obj); err != nil {
		return "<unknown>"
	}

	if obj.Name != "" {
		return obj.Name
	}
	if obj.GenerateName != "" {
		return obj.GenerateName + "<generated>"
	}
	return "<unknown>"
}

func writeResponse(ctx context.Context, mutator Mutator, writer http.ResponseWriter, response *admissionv1.AdmissionResponse) {
	resp, err := json.Marshal(admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: response,
	})
	if err != nil {
		mutator.Errorf(ctx, err, "unable to serialize response")
		metrics.InternalError.WithLabelValues("mutating", mutator.Resource()).Inc()
		writer.WriteHeader(http.StatusInternalServerError)
	}
	if _, err := writer.Write(resp); err != nil {
		mutator.Errorf(ctx, err, "unable to write response")
	}
}

func errorResponse(uid types.UID, err error) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Allowed: false,
		UID:     uid,
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}
