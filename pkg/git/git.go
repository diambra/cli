package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const RefPrefix = "ref: "

func findGitDir(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("unable to resolve absolute path: %w", err)
	}

	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return gitPath, nil
		}

		parent := filepath.Dir(dir)
		// root
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no .git directory found in parents of %s", dir)
}

func GitHeadSHA(dir string) (string, error) {
	dir, err := findGitDir(dir)
	if err != nil {
		return "", err
	}
	file, err := os.ReadFile(filepath.Join(dir, "HEAD"))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(string(file), RefPrefix) {
		return string(file), nil
	}

	refFile, err := os.ReadFile(filepath.Join(dir, strings.TrimSpace(string(file)[len(RefPrefix):])))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(refFile)), nil
}

func GitHeadSHAShort(dir string, n int) (string, error) {
	if n <= 0 {
		n = 7
	}
	sha, err := GitHeadSHA(dir)
	if err != nil {
		return "", err
	}
	if len(sha) < n {
		return sha, nil
	}
	return sha[:n], nil
}
