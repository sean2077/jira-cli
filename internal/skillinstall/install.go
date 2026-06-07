package skillinstall

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	jiraskill "github.com/sean2077/jira-cli/skills/jira-cli"
)

var ErrTargetExists = errors.New("skill target exists")

type Result struct {
	Skill   string
	Root    string
	Target  string
	DryRun  bool
	Force   bool
	Exists  bool
	Changed bool
}

func Install(root string, force, dryRun bool) (Result, error) {
	root = filepath.Clean(root)
	target := filepath.Join(root, jiraskill.Name)
	exists, err := pathExists(target)
	if err != nil {
		return Result{}, err
	}
	result := Result{
		Skill:  jiraskill.Name,
		Root:   root,
		Target: target,
		DryRun: dryRun,
		Force:  force,
		Exists: exists,
	}
	if dryRun {
		return result, nil
	}
	if exists {
		if !force {
			return result, ErrTargetExists
		}
		if err := os.RemoveAll(target); err != nil {
			return result, fmt.Errorf("remove existing target: %w", err)
		}
	}
	if err := copySkillFS(jiraskill.Files, target); err != nil {
		return result, err
	}
	result.Changed = true
	return result, nil
}

func copySkillFS(source fs.FS, target string) error {
	return fs.WalkDir(source, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == "." {
			return os.MkdirAll(target, 0o755)
		}
		dest := filepath.Join(target, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		data, err := fs.ReadFile(source, path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0o644)
	})
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}
