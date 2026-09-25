package ocr

import (
	"strings"
	"testing"
)

func TestLoadRuntimeConfig(t *testing.T) {
	values := map[string]string{
		"MODE": "release", "FRONTEND_ORIGIN": "https://app.example.com",
		"OCR_CAPABILITY_SECRET": strings.Repeat("s", 32),
		"OCR_REPLAY_TABLE":      "expense-ocr-replay",
	}
	config, err := LoadRuntimeConfig(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.FrontendOrigin != values["FRONTEND_ORIGIN"] || config.ReplayTable != values["OCR_REPLAY_TABLE"] {
		t.Fatalf("unexpected config: %#v", config)
	}
}

func TestLoadRuntimeConfigRejectsInvalidValuesWithoutExposingSecrets(t *testing.T) {
	base := map[string]string{
		"MODE": "release", "FRONTEND_ORIGIN": "https://app.example.com",
		"OCR_CAPABILITY_SECRET": strings.Repeat("s", 32),
		"OCR_REPLAY_TABLE":      "expense-ocr-replay",
	}
	for name, test := range map[string][2]string{
		"mode":   {"MODE", "debug"},
		"origin": {"FRONTEND_ORIGIN", "http://app.example.com"},
		"secret": {"OCR_CAPABILITY_SECRET", "short"},
		"table":  {"OCR_REPLAY_TABLE", ""},
	} {
		t.Run(name, func(t *testing.T) {
			key, value := test[0], test[1]
			values := make(map[string]string, len(base))
			for itemKey, itemValue := range base {
				values[itemKey] = itemValue
			}
			values[key] = value
			_, err := LoadRuntimeConfig(func(item string) (string, bool) {
				result, ok := values[item]
				return result, ok
			})
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("expected %s error, got %v", key, err)
			}
			if value != "" && strings.Contains(err.Error(), value) {
				t.Fatalf("error exposed invalid value: %v", err)
			}
		})
	}
}
