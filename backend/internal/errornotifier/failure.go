package errornotifier

import (
	"errors"
	"fmt"
)

type FailureKind string

const (
	PermanentFailure FailureKind = "permanent"
	RetryableFailure FailureKind = "retryable"
)

type Failure struct {
	Kind      FailureKind
	Operation string
}

func (f *Failure) Error() string {
	return fmt.Sprintf("error notifier %s failed (%s)", f.Operation, f.Kind)
}

func IsRetryable(err error) bool {
	var failure *Failure
	return errors.As(err, &failure) && failure.Kind == RetryableFailure
}

func permanent(operation string) error {
	return &Failure{Kind: PermanentFailure, Operation: operation}
}

func retryable(operation string) error {
	return &Failure{Kind: RetryableFailure, Operation: operation}
}

func NewRetryableFailure(operation string) error {
	return retryable(operation)
}
