package validator

import (
	"context"
	"fmt"
)

type Request struct {
	Obj interface{}
}

type Response struct {
	// Validation when is nil means accept. Accept and Rejectf should be
	// used to create validation.
	Validation *Validation
}

type Validation struct {
	valid   bool
	message string
}

func Accept() *Validation {
	return &Validation{
		valid: true,
	}
}

func Rejectf(format string, a ...interface{}) *Validation {
	return &Validation{
		valid:   false,
		message: fmt.Sprintf(format, a...)}
}

// Interface is the validating handler interface.
type Interface interface {
	Name() string
	Validate(ctx context.Context, req Request) (*Response, error)
}
