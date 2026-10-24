package skill

import (
	"fmt"
	"testing"
	"testing/fstest"

	"expense-tracker/agent/skills"
)

func TestEmbeddedSkillsAndValidation(t *testing.T) {
	t.Chdir(t.TempDir())
	expectedModels := map[string]string{
		"orchestrator": "gemini-3.5-flash-lite",
		"get-balance":  "gemini-3.8-flash",
	}
	for name, model := range expectedModels {
		loaded, err := Load(t.Context(), skills.Files, name)
		if err != nil {
			t.Fatal(err)
		}
		invalidInstructions := loaded.Name != name || loaded.Instructions == ""
		invalidMetadata := len(loaded.Version) != 64 || loaded.ModelPreference != model || loaded.Type != "llm"
		if invalidInstructions || invalidMetadata {
			t.Fatalf("invalid loaded skill: %+v", loaded)
		}
	}
	if _, err := Load(t.Context(), skills.Files, "../orchestrator"); err == nil {
		t.Fatal("unapproved skill accepted")
	}
	for _, content := range []string{
		"missing frontmatter", "---\nname: orchestrator\n---\nbody",
		"---\nname: other\ndescription: description\n---\nbody",
		"---\nname: orchestrator\ndescription: description\n---\n  ",
		"---\nname: orchestrator\ndescription: description\nunknown: value\n---\nbody",
	} {
		files := fstest.MapFS{
			"orchestrator/SKILL.md": &fstest.MapFile{Data: []byte(content)},
		}
		if _, err := Load(t.Context(), files, "orchestrator"); err == nil {
			t.Fatalf("invalid skill accepted: %s", content)
		}
	}
}

func TestDiscoveryRejectsBrokenSkillsAndTooManySkills(t *testing.T) {
	files := fstest.MapFS{}
	for range 33 {
		name := fmt.Sprintf("leaf-%02d", len(files))
		content := fmt.Sprintf("---\nname: %s\ndescription: Test leaf.\n---\nInstructions.\n", name)
		files[name+"/SKILL.md"] = &fstest.MapFile{Data: []byte(content)}
	}
	if _, err := Discover(t.Context(), files); err == nil {
		t.Fatal("unbounded discovery accepted")
	}
	files = fstest.MapFS{
		"broken/SKILL.md": &fstest.MapFile{Data: []byte("broken")},
	}
	if _, err := Discover(t.Context(), files); err == nil {
		t.Fatal("malformed bundled skill ignored")
	}
}
