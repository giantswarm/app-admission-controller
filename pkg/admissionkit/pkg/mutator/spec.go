package mutator

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Request struct {
	Obj metav1.Object
}

type Response struct {
	MutatedObj metav1.Object
}

// Interface is the mutating handler interface.
type Interface interface {
	Name() string
	Mutate(ctx context.Context, req Request) (*Response, error)
}
