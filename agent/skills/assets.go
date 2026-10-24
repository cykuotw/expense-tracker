// Package skills contains approved, versioned application skills.
package skills

import "embed"

// Files travels with the executable, independently of its working directory.
//
//go:embed */SKILL.md
var Files embed.FS
