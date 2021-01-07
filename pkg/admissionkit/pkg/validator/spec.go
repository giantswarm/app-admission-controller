package validator

import (
	"context"
	"fmt"
)

type Request struct {
	Obj interface{}
}

type Response struct {
	// Rejection when not nil will reject the validation request. It should
	// be created with Rejectf. When set to nil the request is accepted by
	// the validator.
	Rejection *Rejection
}

type Rejection struct {
	message string
}

func Rejectf(format string, a ...interface{}) *Rejection {
	return &Rejection{
		message: fmt.Sprintf(format, a...),
	}
}

// Interface is the validating handler interface.
type Interface interface {
	Name() string
	Validate(ctx context.Context, req Request) (*Response, error)
}
