// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package marker

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ComputeMarker(rootDir string) (marker string, err error) {

	var trackedHash string
	if trackedHash, err = computeTrackedFilesHash(rootDir); err != nil {
		return "", fmt.Errorf("failed to compute tracked files hash: %w", err)
	}

	var modifiedHash string
	if modifiedHash, err = computeModifiedFilesHash(rootDir); err != nil {
		return "", fmt.Errorf("failed to compute modified files hash: %w", err)
	}

	var untrackedHash string
	if untrackedHash, err = computeUntrackedFilesHash(rootDir); err != nil {
		return "", fmt.Errorf("failed to compute untracked files hash: %w", err)
	}

	var deletedHash string
	if deletedHash, err = computeDeletedFilesHash(rootDir); err != nil {
		return "", fmt.Errorf("failed to compute deleted files hash: %w", err)
	}

	// 5. Собрать финальный маркер
	marker = computeFinalMarker(trackedHash, modifiedHash, untrackedHash, deletedHash)

	return marker, nil
}

func computeFinalMarker(trackedHash string, modifiedHash string, untrackedHash string, deletedHash string) (marker string) {

	components := []string{
		trackedHash,
		modifiedHash,
		untrackedHash,
		deletedHash,
	}

	combined := strings.Join(components, "\n")
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%x", hash)
}

func findGitDir(startDir string) (gitDir string, err error) {

	dir := startDir
	for {
		gitPath := filepath.Join(dir, ".git")
		var info os.FileInfo
		if info, err = os.Stat(gitPath); err == nil {
			if info.IsDir() {
				gitDir = gitPath
				return
			}
			var content []byte
			if content, err = os.ReadFile(gitPath); err == nil {
				gitDir = strings.TrimSpace(string(content))
				if strings.HasPrefix(gitDir, "gitdir: ") {
					gitDir = strings.TrimPrefix(gitDir, "gitdir: ")
					// Если путь относительный, делаем его абсолютным
					if !filepath.IsAbs(gitDir) {
						gitDir = filepath.Join(dir, gitDir)
					}
					return
				}
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			err = fmt.Errorf("git repository not found")
			return
		}
		dir = parent
	}
}

func isGoFile(path string) bool {

	return strings.HasSuffix(strings.ToLower(path), ".go")
}
