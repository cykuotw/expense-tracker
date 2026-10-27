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

func TestExplicitLocalStartup(t *testing.T) {
	t.Setenv("AGENT_MODE", "local")
	t.Setenv("AGENT_WORKER_URL", "http://127.0.0.1:8000")
	if err := validateStartup(nil); err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"", "https://remote.test", "http://0.0.0.0:8000"} {
		t.Setenv("AGENT_WORKER_URL", endpoint)
		if validateStartup(nil) == nil {
			t.Fatal("accepted non-local Worker")
		}
	}
	t.Setenv("AGENT_MODE", "unknown")
	if validateStartup(nil) == nil {
		t.Fatal("accepted unknown mode")
	}
}
