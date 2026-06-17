package install

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

//go:embed skill/*
var skillFS embed.FS

type Runtime struct {
	Name string
	Path string
}

var knownRuntimes = []Runtime{
	{Name: "Claude Code", Path: "~/.claude/skills/frizzle"},
	{Name: "Codex", Path: "~/.codex/skills/frizzle"},
	{Name: "OpenCode CLI", Path: "~/.agents/skills/frizzle"},
	{Name: "Cursor", Path: "~/.cursor/skills/frizzle"},
	{Name: "Windsurf", Path: "~/.windsurf/skills/frizzle"},
	{Name: "Cline", Path: "~/.cline/skills/frizzle"},
	{Name: "Gemini CLI", Path: "~/.gemini/skills/frizzle"},
	{Name: "Amazon Q Developer", Path: "~/.amazonq/skills/frizzle"},
	{Name: "Goose", Path: "~/.goose/skills/frizzle"},
	{Name: "Hermes", Path: "~/.hermes/skills/frizzle"},
	{Name: "OpenClaw", Path: "~/.openclaw/skills/frizzle"},
}

func TryInstall() error {
	if !isTTY() {
		return nil
	}

	install := false
	promptInstall := &survey.Confirm{
		Message: "Would you like to install the frizzle agent skill?",
		Default: true,
	}
	if err := survey.AskOne(promptInstall, &install); err != nil {
		return nil
	}
	if !install {
		return nil
	}

	runtimeLabels := make([]string, len(knownRuntimes))
	defaultSelection := make([]string, 0, len(knownRuntimes))
	for i, r := range knownRuntimes {
		runtimeLabels[i] = fmt.Sprintf("%s  (%s)", r.Name, r.Path)
		if r.exists() {
			defaultSelection = append(defaultSelection, runtimeLabels[i])
		}
	}

	var selected []string
	promptRuntime := &survey.MultiSelect{
		Message: "Select agent runtimes (space to toggle, enter to confirm):",
		Options: runtimeLabels,
		Default: defaultSelection,
	}
	if err := survey.AskOne(promptRuntime, &selected); err != nil {
		return nil
	}

	if len(selected) == 0 {
		fmt.Println("No runtimes selected — skipping skill install")
		return nil
	}

	var installed []string
	for _, label := range selected {
		idx := indexOf(runtimeLabels, label)
		if idx < 0 {
			continue
		}
		r := knownRuntimes[idx]
		targetPath := expandHome(r.Path)
		if err := installSkill(targetPath); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", r.Name, err)
			continue
		}
		installed = append(installed, fmt.Sprintf("  ✓ %s → %s", r.Name, targetPath))
	}

	fmt.Println()
	for _, line := range installed {
		fmt.Println(line)
	}
	return nil
}

func (r Runtime) exists() bool {
	targetPath := expandHome(r.Path)
	_, err := os.Stat(targetPath)
	return err == nil
}

func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func installSkill(targetPath string) error {
	if err := os.RemoveAll(targetPath); err != nil {
		return err
	}
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return err
	}
	return copyEmbedDir("skill", targetPath)
}

func copyEmbedDir(srcDir, dstDir string) error {
	entries, err := skillFS.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}
			if err := copyEmbedDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyEmbedFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyEmbedFile(srcPath, dstPath string) error {
	src, err := skillFS.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}
