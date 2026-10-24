package models

import (
	"fmt"
	"slices"
)

const DefaultModel = "gemini-3.8-flash"
const RouterModel = "gemini-3.5-flash-lite"

// These are application-approved preferences, not a claim of provider availability.
var approved = []string{DefaultModel, RouterModel}

type Resolution struct {
	Requested string
	Resolved  string
	Fallback  bool
	Reason    string
}

func Resolve(preference, defaultModel string) (Resolution, error) {
	if !slices.Contains(approved, defaultModel) {
		return Resolution{}, fmt.Errorf("unapproved default model")
	}
	resolution := Resolution{
		Requested: preference,
		Resolved:  preference,
	}
	if !slices.Contains(approved, preference) {
		resolution.Resolved = defaultModel
		resolution.Fallback = true
		resolution.Reason = "missing_or_unapproved_preference"
	}
	return resolution, nil
}
