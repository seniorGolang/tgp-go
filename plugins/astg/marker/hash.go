// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
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

// computeTrackedFilesHash вычисляет hash отслеживаемых .go файлов.
func computeTrackedFilesHash(rootDir string) (hash string, err error) {

	var files []trackedFile
	if files, err = getGitTrackedFiles(rootDir); err != nil {
		return "", err
	}

	if len(files) == 0 {
		return computeSHA256(""), nil
	}

	// Сортируем для детерминированности
	sort.Slice(files, func(i, j int) bool {
		return files[i].path < files[j].path
	})

	// Формируем строку: path1:hash1\npath2:hash2\n...
	var builder strings.Builder
	for _, file := range files {
		builder.WriteString(file.path)
		builder.WriteString(":")
		builder.WriteString(file.hash)
		builder.WriteString("\n")
	}

	return computeSHA256(builder.String()), nil
}

// computeModifiedFilesHash вычисляет hash измененных файлов.
func computeModifiedFilesHash(rootDir string) (hash string, err error) {

	var files []string
	if files, err = getGitModifiedFiles(rootDir); err != nil {
		return "", err
	}

	if len(files) == 0 {
		return computeSHA256(""), nil
	}

	// Сортируем для детерминированности
	sort.Strings(files)

	// Вычисляем hash для каждого файла
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

	// Формируем строку: path1:hash1\npath2:hash2\n...
	var builder strings.Builder
	for _, fh := range fileHashes {
		builder.WriteString(fh.path)
		builder.WriteString(":")
		builder.WriteString(fh.hash)
		builder.WriteString("\n")
	}

	return computeSHA256(builder.String()), nil
}

// computeUntrackedFilesHash вычисляет hash неотслеживаемых .go файлов.
func computeUntrackedFilesHash(rootDir string) (hash string, err error) {

	var files []string
	if files, err = getGitUntrackedFiles(rootDir); err != nil {
		return "", err
	}

	if len(files) == 0 {
		return computeSHA256(""), nil
	}

	// Сортируем для детерминированности
	sort.Strings(files)

	// Вычисляем hash для каждого файла
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

	// Формируем строку: path1:hash1\npath2:hash2\n...
	var builder strings.Builder
	for _, fh := range fileHashes {
		builder.WriteString(fh.path)
		builder.WriteString(":")
		builder.WriteString(fh.hash)
		builder.WriteString("\n")
	}

	return computeSHA256(builder.String()), nil
}

// computeDeletedFilesHash вычисляет hash списка удаленных файлов.
func computeDeletedFilesHash(rootDir string) (hash string, err error) {

	var files []string
	if files, err = getGitDeletedFiles(rootDir); err != nil {
		return "", err
	}

	if len(files) == 0 {
		return computeSHA256(""), nil
	}

	// Сортируем для детерминированности
	sort.Strings(files)

	// Формируем строку: path1\npath2\n...
	combined := strings.Join(files, "\n")
	return computeSHA256(combined), nil
}

// computeSHA256 вычисляет SHA256 hash строки.
func computeSHA256(data string) string {

	hasher := sha256.New()
	hasher.Write([]byte(data))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// computeFileHash вычисляет hash файла через SHA256.
func computeFileHash(rootDir string, filePath string) (hash string, err error) {

	absPath := filepath.Join(rootDir, filePath)
	return computeFileSHA256(absPath)
}

// computeFileSHA256 вычисляет SHA256 hash файла.
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
