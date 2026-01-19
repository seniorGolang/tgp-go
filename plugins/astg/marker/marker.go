// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package marker

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MarkerVersion версия формата маркера.
	MarkerVersion = "v1"
)

// ComputeMarker вычисляет маркер состояния проекта.
// Маркер - это SHA256 hash, который уникально идентифицирует состояние всех релевантных файлов проекта.
// rootDir - корневая директория проекта (обычно internal.ProjectRoot).
func ComputeMarker(rootDir string) (marker string, err error) {

	// Проверяем, что это Git репозиторий
	var gitDir string
	if gitDir, err = findGitDir(rootDir); err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}

	// 1. Получить текущий коммит (может быть пустым для пустого репозитория)
	var commitHash string
	if commitHash, err = getGitCommitHash(gitDir); err != nil {
		return "", fmt.Errorf("failed to get git commit: %w", err)
	}
	// Если коммит пустой (пустой репозиторий), используем специальное значение
	if commitHash == "" {
		commitHash = "empty-repository"
	}

	// 2. Получить hash отслеживаемых .go файлов
	var trackedHash string
	if trackedHash, err = computeTrackedFilesHash(rootDir); err != nil {
		return "", fmt.Errorf("failed to compute tracked files hash: %w", err)
	}

	// 3. Получить hash измененных файлов
	var modifiedHash string
	if modifiedHash, err = computeModifiedFilesHash(rootDir); err != nil {
		return "", fmt.Errorf("failed to compute modified files hash: %w", err)
	}

	// 4. Получить hash неотслеживаемых .go файлов
	var untrackedHash string
	if untrackedHash, err = computeUntrackedFilesHash(rootDir); err != nil {
		return "", fmt.Errorf("failed to compute untracked files hash: %w", err)
	}

	// 5. Получить hash списка удаленных файлов
	var deletedHash string
	if deletedHash, err = computeDeletedFilesHash(rootDir); err != nil {
		return "", fmt.Errorf("failed to compute deleted files hash: %w", err)
	}

	// 6. Собрать финальный маркер
	marker = computeFinalMarker(commitHash, trackedHash, modifiedHash, untrackedHash, deletedHash)

	return marker, nil
}

// computeFinalMarker вычисляет финальный маркер из всех компонентов.
func computeFinalMarker(commitHash string, trackedHash string, modifiedHash string, untrackedHash string, deletedHash string) (marker string) {

	components := []string{
		MarkerVersion,
		commitHash,
		trackedHash,
		modifiedHash,
		untrackedHash,
		deletedHash,
	}

	combined := strings.Join(components, "\n")
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%x", hash)
}

// findGitDir находит директорию .git репозитория.
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
			// Если .git - это файл (submodule), читаем его содержимое
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

// isGoFile проверяет, является ли файл Go файлом.
func isGoFile(path string) bool {

	return strings.HasSuffix(strings.ToLower(path), ".go")
}
