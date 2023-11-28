package validator

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/giantswarm/microerror"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-admission-controller/pkg/metrics"
)

type Validator interface {
	Debugf(ctx context.Context, format string, params ...interface{})
	Errorf(ctx context.Context, err error, format string, params ...interface{})
	Resource() string
	Validate(review *admissionv1.AdmissionRequest) (bool, error)
}

var (
	scheme       = runtime.NewScheme()
	codecs       = serializer.NewCodecFactory(scheme)
	Deserializer = codecs.UniversalDeserializer()
)

func Handler(validator Validator) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ctx := context.Background()
		start := time.Now()
		defer func() {
			metrics.DurationRequests.WithLabelValues("validating", validator.Resource()).Observe(float64(time.Since(start)) / float64(time.Second))
		}()

		if request.Header.Get("Content-Type") != "application/json" {
			validator.Errorf(ctx, nil, "invalid content-type: %s", request.Header.Get("Content-Type"))
			metrics.InvalidRequests.WithLabelValues("validating", validator.Resource()).Inc()
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		data, err := io.ReadAll(request.Body)
		if err != nil {
			validator.Errorf(ctx, err, "unable to read request")
			metrics.InternalError.WithLabelValues("validating", validator.Resource()).Inc()
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		review := admissionv1.AdmissionReview{}
		if _, _, err := Deserializer.Decode(data, nil, &review); err != nil {
			validator.Errorf(ctx, err, "unable to parse admission review request")
			metrics.InvalidRequests.WithLabelValues("validating", validator.Resource()).Inc()
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		allowed, err := validator.Validate(review.Request)
		if err != nil {
			writeResponse(ctx, validator, writer, errorResponse(review.Request.UID, microerror.Mask(err)))
			metrics.RejectedRequests.WithLabelValues("validating", validator.Resource()).Inc()
			return
		}

		metrics.SuccessfulRequests.WithLabelValues("validating", validator.Resource()).Inc()

		writeResponse(ctx, validator, writer, &admissionv1.AdmissionResponse{
			Allowed: allowed,
			UID:     review.Request.UID,
		})
	}
}

func writeResponse(ctx context.Context, validator Validator, writer http.ResponseWriter, response *admissionv1.AdmissionResponse) {
	resp, err := json.Marshal(admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: response,
	})
	if err != nil {
		validator.Errorf(ctx, err, "unable to serialize response")
		metrics.InternalError.WithLabelValues("validating", validator.Resource()).Inc()
		writer.WriteHeader(http.StatusInternalServerError)
	}

	if _, err := writer.Write(resp); err != nil {
		validator.Errorf(ctx, err, "unable to write response")
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
