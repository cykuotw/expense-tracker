package api_test

import (
	"expense-tracker/agent/internal/api"
	"testing"
)

func TestNewLocalHandlerRejectsNilApplication(t *testing.T) {
	if _, err := api.NewLocalHandler(nil); err == nil {
		t.Fatal("accepted nil application")
	}
}
