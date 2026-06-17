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
	{Name: "Codex (OpenCode)", Path: "~/.codex/skills/frizzle"},
	{Name: "OpenCode (agents)", Path: "~/.agents/skills/frizzle"},
	{Name: "I'll specify the directory", Path: ""},
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

	runtimeNames := make([]string, len(knownRuntimes))
	for i, r := range knownRuntimes {
		runtimeNames[i] = r.Name
	}

	var selected int
	promptRuntime := &survey.Select{
		Message: "Which agent runtime are you using?",
		Options: runtimeNames,
		Default: runtimeNames[0],
	}
	if err := survey.AskOne(promptRuntime, &selected); err != nil {
		return nil
	}

	targetPath := knownRuntimes[selected].Path
	if targetPath == "" {
		promptPath := &survey.Input{
			Message: "Directory to install the skill into:",
			Suggest: func(toComplete string) []string {
				return nil
			},
		}
		if err := survey.AskOne(promptPath, &targetPath); err != nil {
			return nil
		}
	}

	targetPath = expandHome(targetPath)

	if err := installSkill(targetPath); err != nil {
		return fmt.Errorf("failed to install skill: %w", err)
	}

	fmt.Printf("\n✓ Installed frizzle skill to %s\n", targetPath)
	return nil
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
