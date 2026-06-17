package install

import (
	"os"
	"testing"
)

func TestInstallSkill(t *testing.T) {
	target := t.TempDir() + "/skills/frizzle"

	if err := installSkill(target); err != nil {
		t.Fatalf("installSkill: %v", err)
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	foundSKILL := false
	foundRefs := false
	for _, e := range entries {
		if e.Name() == "SKILL.md" {
			foundSKILL = true
		}
		if e.Name() == "references" && e.IsDir() {
			foundRefs = true
		}
	}

	if !foundSKILL {
		t.Error("SKILL.md not found")
	}
	if !foundRefs {
		t.Error("references/ not found")
	}
}
