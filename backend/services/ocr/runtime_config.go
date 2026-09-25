package ocr

import (
	"fmt"
	"net/url"
	"strings"
)

type RuntimeConfig struct {
	Mode             string
	FrontendOrigin   string
	CapabilitySecret []byte
	ReplayTable      string
}

func LoadRuntimeConfig(lookup func(string) (string, bool)) (RuntimeConfig, error) {
	read := func(key string) (string, error) {
		value, ok := lookup(key)
		value = strings.TrimSpace(value)
		if !ok || value == "" {
			return "", fmt.Errorf("%s must be configured", key)
		}
		return value, nil
	}

	mode, err := read("MODE")
	if err != nil {
		return RuntimeConfig{}, err
	}
	if mode != "release" && mode != "test" {
		return RuntimeConfig{}, fmt.Errorf("MODE must be release or test")
	}
	origin, err := read("FRONTEND_ORIGIN")
	if err != nil {
		return RuntimeConfig{}, err
	}
	parsedOrigin, err := url.Parse(origin)
	if err != nil || parsedOrigin.Scheme == "" || parsedOrigin.Host == "" || parsedOrigin.Path != "" ||
		parsedOrigin.RawQuery != "" || parsedOrigin.Fragment != "" || parsedOrigin.User != nil ||
		(mode == "release" && parsedOrigin.Scheme != "https") {
		return RuntimeConfig{}, fmt.Errorf("FRONTEND_ORIGIN must be an exact origin")
	}
	secret, err := read("OCR_CAPABILITY_SECRET")
	if err != nil {
		return RuntimeConfig{}, err
	}
	if len([]byte(secret)) < 32 || strings.HasPrefix(strings.ToUpper(secret), "REPLACE") {
		return RuntimeConfig{}, fmt.Errorf("OCR_CAPABILITY_SECRET must be a non-placeholder secret of at least 32 bytes")
	}
	table, err := read("OCR_REPLAY_TABLE")
	if err != nil {
		return RuntimeConfig{}, err
	}
	return RuntimeConfig{
		Mode:             mode,
		FrontendOrigin:   origin,
		CapabilitySecret: []byte(secret),
		ReplayTable:      table,
	}, nil
}
