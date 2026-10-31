// Package skill loads approved skills and validates their registry and tool grants.
package skill

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"strings"

	adkskill "google.golang.org/adk/v2/tool/skilltoolset/skill"
)

const (
	ModelKey         = "model"
	maxSkillBytes    = 32 << 10
	maxBundledSkills = 32
)

type Skill struct {
	Name         string
	Description  string
	Instructions string
	Version      string
	ModelRole    string
	Type         string
	Tools        []string
}

// Load reads a skill from trusted bundled assets. Tool requests are validated
// against the runtime catalog before agent construction.
func Load(ctx context.Context, files fs.FS, name string) (Skill, error) {
	if !fs.ValidPath(name) || strings.Contains(name, "/") {
		return Skill{}, fmt.Errorf("invalid skill directory %q", name)
	}
	content, err := fs.ReadFile(files, name+"/SKILL.md")
	if err != nil {
		return Skill{}, fmt.Errorf("read approved skill: %w", err)
	}
	if len(content) > maxSkillBytes {
		return Skill{}, fmt.Errorf("skill exceeds 32 KiB")
	}
	source := adkskill.NewFileSystemSource(files)
	frontmatter, err := source.LoadFrontmatter(ctx, name)
	if err != nil {
		return Skill{}, err
	}
	instructions, err := source.LoadInstructions(ctx, name)
	if err != nil {
		return Skill{}, err
	}
	if strings.TrimSpace(instructions) == "" {
		return Skill{}, fmt.Errorf("empty skill instructions")
	}
	agentType := frontmatter.Metadata["type"]
	if agentType == "" {
		agentType = "llm"
	}
	if agentType != "llm" {
		return Skill{}, fmt.Errorf("unsupported agent type %q in %s", agentType, name)
	}
	if _, exists := frontmatter.Metadata["mode"]; exists {
		return Skill{}, fmt.Errorf("agent mode is runtime-owned: %s", name)
	}
	if len(frontmatter.AllowedTools) > 0 {
		return Skill{}, fmt.Errorf("use metadata.allowed-tools in %s", name)
	}
	return Skill{
		Name:         frontmatter.Name,
		Description:  frontmatter.Description,
		Instructions: instructions,
		Version:      fmt.Sprintf("%x", sha256.Sum256(content)),
		ModelRole:    frontmatter.Metadata[ModelKey],
		Type:         agentType,
		Tools:        strings.Fields(frontmatter.Metadata["allowed-tools"]),
	}, nil
}

// Discover scans immediate skill directories in a trusted release filesystem.
// It does not accept user-selected paths or reload files during requests.
func Discover(ctx context.Context, files fs.FS) ([]Skill, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, err
	}
	var result []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if len(result) >= maxBundledSkills {
			return nil, fmt.Errorf("too many bundled skills")
		}
		loaded, err := Load(ctx, files, entry.Name())
		if err != nil {
			return nil, err
		}
		result = append(result, loaded)
	}
	return result, nil
}
