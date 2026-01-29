// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package marker

import (
	"crypto/sha1" //nolint:gosec // SHA1 используется для совместимости с форматом Git
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type trackedFile struct {
	path string
	hash string
}

type indexEntryWithHash struct {
	path  string
	hash  string
	size  int64
	mtime int64
}

func parseGitIndexWithHash(indexPath string) (entries []indexEntryWithHash, err error) {

	var data []byte
	if data, err = os.ReadFile(indexPath); err != nil {
		return
	}

	if len(data) < 12 {
		err = fmt.Errorf("index file too short")
		return
	}

	if string(data[0:4]) != "DIRC" {
		err = fmt.Errorf("invalid index signature")
		return
	}

	// Читаем количество записей (4 bytes после signature)
	entryCount := int(data[8])<<24 | int(data[9])<<16 | int(data[10])<<8 | int(data[11])

	entries = make([]indexEntryWithHash, 0)
	offset := 12

	for i := 0; i < entryCount && offset < len(data); i++ {
		if offset+62 > len(data) {
			break // Недостаточно данных для entry
		}

		// Git index entry format:
		// - ctime (8 bytes: 4 bytes seconds + 4 bytes nanoseconds)
		// - mtime (8 bytes: 4 bytes seconds + 4 bytes nanoseconds)
		// - dev (4 bytes)
		// - ino (4 bytes)
		// - mode (4 bytes)
		// - uid (4 bytes)
		// - gid (4 bytes)
		// - size (4 bytes)
		// - sha (20 bytes) - начинается с offset+40
		// - flags (2 bytes)
		// - path (variable, null-terminated, padded to multiple of 8)

		// Читаем mtime (секунды)
		mtimeSeconds := int64(data[offset+8])<<24 | int64(data[offset+9])<<16 | int64(data[offset+10])<<8 | int64(data[offset+11])

		// Читаем size
		size := int64(data[offset+36])<<24 | int64(data[offset+37])<<16 | int64(data[offset+38])<<8 | int64(data[offset+39])

		// Читаем SHA (20 bytes для SHA-1)
		shaOffset := offset + 40
		if shaOffset+20 > len(data) {
			break
		}
		shaBytes := data[shaOffset : shaOffset+20]
		shaHash := hex.EncodeToString(shaBytes)

		// Читаем flags (2 bytes после sha)
		flagsOffset := offset + 60
		if flagsOffset+2 > len(data) {
			break
		}
		flags := int(data[flagsOffset])<<8 | int(data[flagsOffset+1])
		nameLen := flags & 0xFFF // Младшие 12 бит - длина имени

		// Читаем путь
		pathOffset := offset + 62
		if pathOffset+nameLen > len(data) {
			break
		}
		path := string(data[pathOffset : pathOffset+nameLen])
		path = strings.TrimRight(path, "\x00") // Удаляем null-terminator

		entries = append(entries, indexEntryWithHash{
			path:  path,
			hash:  shaHash,
			size:  size,
			mtime: mtimeSeconds,
		})

		// Следующая entry начинается после padding (кратно 8)
		entryLen := 62 + nameLen
		padding := (8 - (entryLen % 8)) % 8
		offset += entryLen + padding
	}

	return
}

func getGitTrackedFiles(rootDir string) (files []trackedFile, err error) {

	var gitDir string
	if gitDir, err = findGitDir(rootDir); err != nil {
		return nil, fmt.Errorf("failed to find git dir: %w", err)
	}

	indexPath := filepath.Join(gitDir, "index")
	var statErr error
	if _, statErr = os.Stat(indexPath); statErr != nil {
		// Если индекс не существует, возвращаем пустой список
		return []trackedFile{}, nil
	}

	var entries []indexEntryWithHash
	if entries, err = parseGitIndexWithHash(indexPath); err != nil {
		return nil, fmt.Errorf("failed to parse git index: %w", err)
	}

	files = make([]trackedFile, 0)
	for _, entry := range entries {
		// Фильтруем только .go файлы
		if isGoFile(entry.path) {
			files = append(files, trackedFile{
				path: entry.path,
				hash: entry.hash,
			})
		}
	}

	return files, nil
}

func getGitModifiedFiles(rootDir string) (files []string, err error) {

	var gitDir string
	if gitDir, err = findGitDir(rootDir); err != nil {
		return nil, fmt.Errorf("failed to find git dir: %w", err)
	}

	indexPath := filepath.Join(gitDir, "index")
	var statErr error
	if _, statErr = os.Stat(indexPath); statErr != nil {
		// Если индекс не существует, возвращаем пустой список
		return []string{}, nil
	}

	var indexEntries []indexEntryWithHash
	if indexEntries, err = parseGitIndexWithHash(indexPath); err != nil {
		return nil, fmt.Errorf("failed to parse git index: %w", err)
	}

	fileMap := make(map[string]bool)

	for _, entry := range indexEntries {
		filePath := filepath.Join(rootDir, entry.path)
		var fileInfo os.FileInfo
		if fileInfo, err = os.Stat(filePath); err != nil {
			// Файл удален из рабочей директории - это не изменение, а удаление
			continue
		}

		if fileInfo.Size() != entry.size || fileInfo.ModTime().Unix() != entry.mtime {
			var currentHash string
			if currentHash, err = computeFileSHA1(filePath); err != nil {
				continue
			}

			// Если hash отличается, файл изменен
			if currentHash != entry.hash {
				if isGoFile(entry.path) {
					fileMap[entry.path] = true
				}
			}
		}
	}

	files = make([]string, 0, len(fileMap))
	for file := range fileMap {
		files = append(files, file)
	}

	return files, nil
}

func computeFileSHA1(filePath string) (hash string, err error) {

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var fileInfo os.FileInfo
	if fileInfo, err = file.Stat(); err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	size := fileInfo.Size()
	header := fmt.Sprintf("blob %d\x00", size)

	hasher := sha1.New() //nolint:gosec // SHA1 используется для совместимости с форматом Git
	hasher.Write([]byte(header))

	// Копируем содержимое файла
	buf := make([]byte, 32*1024)
	for {
		var n int
		n, err = file.Read(buf)
		if n > 0 {
			hasher.Write(buf[:n])
		}
		if err != nil {
			// EOF - нормальное завершение чтения
			if err == io.EOF {
				err = nil
			}
			break
		}
	}

	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	hash = hex.EncodeToString(hasher.Sum(nil))
	return hash, nil
}

func getGitUntrackedFiles(rootDir string) (files []string, err error) {

	var gitDir string
	if gitDir, err = findGitDir(rootDir); err != nil {
		return nil, fmt.Errorf("failed to find git dir: %w", err)
	}

	indexPath := filepath.Join(gitDir, "index")
	var indexEntries []indexEntryWithHash
	if _, statErr := os.Stat(indexPath); statErr == nil {
		if indexEntries, err = parseGitIndexWithHash(indexPath); err != nil {
			return nil, fmt.Errorf("failed to parse git index: %w", err)
		}
	}

	// Создаем карту отслеживаемых файлов
	trackedFiles := make(map[string]bool)
	for _, entry := range indexEntries {
		trackedFiles[entry.path] = true
	}

	var ignorePatterns []string
	gitignorePath := filepath.Join(rootDir, ".gitignore")
	if gitignoreContent, readErr := os.ReadFile(gitignorePath); readErr == nil {
		ignorePatterns = parseGitignore(string(gitignoreContent))
	}

	files = make([]string, 0)

	// Сканируем рабочую директорию
	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		var relPath string
		if relPath, walkErr = filepath.Rel(rootDir, path); walkErr != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		if trackedFiles[relPath] {
			return nil
		}

		if isIgnored(relPath, ignorePatterns) {
			return nil
		}

		// Фильтруем только .go файлы
		if isGoFile(relPath) {
			files = append(files, relPath)
		}

		return nil
	})

	return files, err
}

func parseGitignore(content string) (patterns []string) {

	lines := strings.Split(content, "\n")
	patterns = make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	return patterns
}

func isIgnored(path string, patterns []string) bool {

	for _, pattern := range patterns {
		if matchesGitignorePattern(path, pattern) {
			return true
		}
	}

	return false
}

func matchesGitignorePattern(path string, pattern string) bool {

	// Упрощенная реализация - поддерживаем базовые паттерны
	// Полная реализация .gitignore очень сложная, но для большинства случаев достаточно

	// Если паттерн заканчивается на /, это директория
	if strings.HasSuffix(pattern, "/") {
		pattern = strings.TrimSuffix(pattern, "/")
		if strings.HasPrefix(path, pattern+"/") || path == pattern {
			return true
		}
		return false
	}

	// Если паттерн начинается с /, это абсолютный путь от корня
	if strings.HasPrefix(pattern, "/") {
		pattern = strings.TrimPrefix(pattern, "/")
		return path == pattern || strings.HasPrefix(path, pattern+"/")
	}

	// Паттерн может содержать * для любого количества символов
	if strings.Contains(pattern, "*") {
		// Простая реализация для * в конце или начале
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			return strings.HasPrefix(path, prefix)
		}
		if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			return strings.HasSuffix(path, suffix)
		}
		// Для более сложных случаев используем простую проверку подстроки
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(path, parts[0]) && strings.HasSuffix(path, parts[1])
		}
	}

	// Обычное совпадение
	return path == pattern || strings.HasPrefix(path, pattern+"/")
}

func getGitDeletedFiles(rootDir string) (files []string, err error) {

	var gitDir string
	if gitDir, err = findGitDir(rootDir); err != nil {
		return nil, fmt.Errorf("failed to find git dir: %w", err)
	}

	indexPath := filepath.Join(gitDir, "index")
	var statErr error
	if _, statErr = os.Stat(indexPath); statErr != nil {
		// Если индекс не существует, возвращаем пустой список
		return []string{}, nil
	}

	var indexEntries []indexEntryWithHash
	if indexEntries, err = parseGitIndexWithHash(indexPath); err != nil {
		return nil, fmt.Errorf("failed to parse git index: %w", err)
	}

	files = make([]string, 0)

	for _, entry := range indexEntries {
		filePath := filepath.Join(rootDir, entry.path)
		if _, statErr = os.Stat(filePath); statErr != nil {
			if os.IsNotExist(statErr) {
				// Файл существует в индексе, но отсутствует в рабочей директории
				files = append(files, entry.path)
			}
		}
	}

	return files, nil
}
