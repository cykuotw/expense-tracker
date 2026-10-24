package main

import "testing"

func TestFixtureStartupIsolation(t *testing.T) {
	if err := validateStartup(nil); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_LAMBDA_RUNTIME_API", "synthetic")
	if validateStartup(nil) == nil {
		t.Fatal("fixture accepted in Lambda")
	}
}
