// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package marker

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func computeTrackedFilesHash(rootDir string) (hash string, err error) {

	var files []trackedFile
	if files, err = getGitTrackedFiles(rootDir); err != nil {
		return "", err
	}

	if len(files) == 0 {
		return computeSHA256(""), nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].path < files[j].path
	})

	var builder strings.Builder
	for _, file := range files {
		builder.WriteString(file.path)
		builder.WriteString(":")
		builder.WriteString(file.hash)
		builder.WriteString("\n")
	}

	return computeSHA256(builder.String()), nil
}

func computeModifiedFilesHash(rootDir string) (hash string, err error) {

	var files []string
	if files, err = getGitModifiedFiles(rootDir); err != nil {
		return "", err
	}

	if len(files) == 0 {
		return computeSHA256(""), nil
	}

	sort.Strings(files)

	type fileHash struct {
		path string
		hash string
	}

	fileHashes := make([]fileHash, 0, len(files))
	for _, file := range files {
		var fileHashValue string
		if fileHashValue, err = computeFileHash(rootDir, file); err != nil {
			// Если не удалось вычислить hash файла, пропускаем его
			// Это может произойти, если файл был удален после git diff
			continue
		}

		fileHashes = append(fileHashes, fileHash{
			path: file,
			hash: fileHashValue,
		})
	}

	if len(fileHashes) == 0 {
		return computeSHA256(""), nil
	}

	var builder strings.Builder
	for _, fh := range fileHashes {
		builder.WriteString(fh.path)
		builder.WriteString(":")
		builder.WriteString(fh.hash)
		builder.WriteString("\n")
	}

	return computeSHA256(builder.String()), nil
}

func computeUntrackedFilesHash(rootDir string) (hash string, err error) {

	var files []string
	if files, err = getGitUntrackedFiles(rootDir); err != nil {
		return "", err
	}

	if len(files) == 0 {
		return computeSHA256(""), nil
	}

	sort.Strings(files)

	type fileHash struct {
		path string
		hash string
	}

	fileHashes := make([]fileHash, 0, len(files))
	for _, file := range files {
		var fileHashValue string
		if fileHashValue, err = computeFileHash(rootDir, file); err != nil {
			// Если не удалось вычислить hash файла, пропускаем его
			continue
		}

		fileHashes = append(fileHashes, fileHash{
			path: file,
			hash: fileHashValue,
		})
	}

	if len(fileHashes) == 0 {
		return computeSHA256(""), nil
	}

	var builder strings.Builder
	for _, fh := range fileHashes {
		builder.WriteString(fh.path)
		builder.WriteString(":")
		builder.WriteString(fh.hash)
		builder.WriteString("\n")
	}

	return computeSHA256(builder.String()), nil
}

func computeDeletedFilesHash(rootDir string) (hash string, err error) {

	var files []string
	if files, err = getGitDeletedFiles(rootDir); err != nil {
		return "", err
	}

	if len(files) == 0 {
		return computeSHA256(""), nil
	}

	sort.Strings(files)

	combined := strings.Join(files, "\n")
	return computeSHA256(combined), nil
}

func computeSHA256(data string) string {

	hasher := sha256.New()
	hasher.Write([]byte(data))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func computeFileHash(rootDir string, filePath string) (hash string, err error) {

	absPath := filepath.Join(rootDir, filePath)
	return computeFileSHA256(absPath)
}

func computeFileSHA256(filePath string) (hash string, err error) {

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err = io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
