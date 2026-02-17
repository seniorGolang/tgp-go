// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package marker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func findGitDir(startDir string) (gitDir string, err error) {

	dir := startDir

	for {
		gitPath := filepath.Join(dir, ".git")
		var info os.FileInfo
		if info, err = os.Stat(gitPath); err == nil {
			if info.IsDir() {
				return gitPath, nil
			}
			var content []byte
			if content, err = os.ReadFile(gitPath); err == nil {
				gitDir = strings.TrimSpace(string(content))
				if strings.HasPrefix(gitDir, "gitdir: ") {
					gitDir = strings.TrimPrefix(gitDir, "gitdir: ")
					if !filepath.IsAbs(gitDir) {
						gitDir = filepath.Join(dir, gitDir)
					}
					return gitDir, nil
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("git repository not found")
		}
		dir = parent
	}
}
