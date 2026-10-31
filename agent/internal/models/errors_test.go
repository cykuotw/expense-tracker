package models

import (
	"errors"
	"fmt"
	"testing"
)

func TestPublicCodePreservesErrorContracts(t *testing.T) {
	for _, test := range []struct {
		err  error
		code Code
	}{
		{ErrMissing, NotFound},
		{ErrConflict, Limit},
		{ErrPayload, Limit},
		{ErrBudget, Limit},
		{ErrStorage, Unavailable},
		{&Error{Code: Unauthenticated}, Unauthenticated},
		{&Error{Code: Forbidden}, Forbidden},
		{&Error{Code: "unsupported"}, Internal},
		{errors.New("private provider detail"), Unavailable},
	} {
		if got := PublicCode(fmt.Errorf("wrapped: %w", test.err)); got != test.code {
			t.Fatalf("got %q, want %q", got, test.code)
		}
	}
}
