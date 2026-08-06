// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"tgp/internal"
	"tgp/internal/model"
)

var errNewFileFound = errors.New("new file found")

func collectGitInfo(project *model.Project) (err error) {

	var gitDir string
	if gitDir, err = findGitDir(internal.ProjectRoot); err != nil {
		return
	}

	repoRoot := filepath.Dir(gitDir)

	gitInfo := &model.GitInfo{}

	var commit string
	if commit, err = getGitCommit(gitDir); err == nil {
		gitInfo.Commit = commit
	}

	var branch string
	if branch, err = getGitBranch(gitDir); err == nil {
		gitInfo.Branch = branch
	}

	var tag string
	if tag, err = getGitTag(gitDir, gitInfo.Commit); err == nil && tag != "" {
		gitInfo.Tag = tag
	}

	var dirty bool
	if dirty, err = isGitDirty(gitDir, repoRoot); err == nil {
		gitInfo.Dirty = dirty
	}

	var user string
	var email string
	if user, email, err = getGitUser(gitDir); err == nil {
		if user != "" {
			gitInfo.User = user
		}
		if email != "" {
			gitInfo.Email = email
		}
	}

	var remoteURL string
	if remoteURL, err = getGitRemoteURL(gitDir); err == nil && remoteURL != "" {
		gitInfo.RemoteURL = remoteURL
	}

	project.Git = gitInfo
	return
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
				if after, ok := strings.CutPrefix(gitDir, "gitdir: "); ok {
					gitDir = after
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
			err = errors.New("git repository not found")
			return
		}
		dir = parent
	}
}

func getGitCommit(gitDir string) (commit string, err error) {

	// Читаем HEAD
	var headContent []byte
	if headContent, err = readGitFile(gitDir, "HEAD"); err != nil {
		return
	}

	headStr := strings.TrimSpace(string(headContent))

	// Если HEAD указывает на ветку (ref: refs/heads/master)
	if after, ok := strings.CutPrefix(headStr, "ref: "); ok {
		refPath := after
		refPath = strings.TrimSpace(refPath)
		var commitBytes []byte
		if commitBytes, err = readGitFile(gitDir, filepath.FromSlash(refPath)); err != nil {
			return
		}
		commit = strings.TrimSpace(string(commitBytes))
		return
	}

	// Если HEAD указывает напрямую на коммит (detached HEAD)
	commit = headStr
	return
}

func getGitBranch(gitDir string) (branch string, err error) {

	headPath := filepath.Join(gitDir, "HEAD")
	var headContent []byte
	if headContent, err = os.ReadFile(headPath); err != nil {
		return
	}

	headStr := strings.TrimSpace(string(headContent))

	// Если HEAD указывает на ветку
	if after, ok := strings.CutPrefix(headStr, "ref: "); ok {
		refPath := after
		refPath = strings.TrimSpace(refPath)
		if after, ok := strings.CutPrefix(refPath, "refs/heads/"); ok {
			branch = after
			return
		}
		// Если это другой тип ref (например, refs/remotes/origin/master), возвращаем пустую строку
		return
	}

	// Detached HEAD - нет ветки
	return
}

func getGitTag(gitDir string, commitHash string) (tag string, err error) {

	if commitHash == "" {
		return
	}

	tagsDir := filepath.Join(gitDir, "refs", "tags")
	var entries []os.DirEntry
	if entries, err = os.ReadDir(tagsDir); err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		var tagContent []byte
		if tagContent, err = readGitFile(gitDir, "refs", "tags", entry.Name()); err != nil {
			continue
		}

		tagHash := strings.TrimSpace(string(tagContent))

		if tagHash == commitHash {
			tag = entry.Name()
			return
		}

		// Аннотированный тег имеет формат: object <hash>\ntype tag\n...
		// Нужно найти объект тега и проверить, на какой коммит он указывает
		if strings.HasPrefix(tagHash, "ref: ") {
			// Это симлинк на другой ref, пропускаем
			continue
		}

		if len(tagHash) < 3 || !isHexHash(tagHash) {
			continue
		}

		var tagObjContent []byte
		if tagObjContent, err = readGitFile(gitDir, "objects", tagHash[:2], tagHash[2:]); err == nil {
			// Парсим объект тега (формат: object <hash>\ntype tag\n...)
			content := string(tagObjContent)
			lines := strings.SplitSeq(content, "\n")
			for line := range lines {
				if after, ok := strings.CutPrefix(line, "object "); ok {
					objHash := strings.TrimSpace(after)
					if objHash == commitHash {
						tag = entry.Name()
						return
					}
					break
				}
			}
		}
	}

	return
}

func readGitFile(gitDir string, pathParts ...string) (content []byte, err error) {

	var basePath string
	if basePath, err = filepath.Abs(filepath.Clean(gitDir)); err != nil {
		return
	}
	targetPath := filepath.Join(append([]string{basePath}, pathParts...)...)
	var targetAbsPath string
	if targetAbsPath, err = filepath.Abs(filepath.Clean(targetPath)); err != nil {
		return
	}
	basePrefix := basePath + string(os.PathSeparator)
	if targetAbsPath != basePath && !strings.HasPrefix(targetAbsPath, basePrefix) {
		return nil, errors.New("unsafe git path")
	}
	return os.ReadFile(targetAbsPath)
}

func isHexHash(value string) (ok bool) {

	if value == "" {
		return
	}
	for _, char := range value {
		isDigit := char >= '0' && char <= '9'
		isLowerHex := char >= 'a' && char <= 'f'
		isUpperHex := char >= 'A' && char <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return
		}
	}
	return true
}

func isGitDirty(gitDir string, repoRoot string) (isDirty bool, err error) {

	indexPath := filepath.Join(gitDir, "index")
	if _, indexStatErr := os.Stat(indexPath); indexStatErr == nil {
		var indexEntries []indexEntry
		if indexEntries, err = parseGitIndex(indexPath); err != nil {
			return
		}

		for _, entry := range indexEntries {
			filePath := filepath.Join(repoRoot, entry.path)
			var fileInfo os.FileInfo
			if fileInfo, err = os.Stat(filePath); err != nil {
				// Файл удален из рабочей директории
				isDirty = true
				return
			}

			// Файл изменен (размер или время модификации)
			if fileInfo.Size() != entry.size || fileInfo.ModTime().Unix() != entry.mtime {
				isDirty = true
				return
			}
		}

		// Это упрощенная проверка - не учитывает .gitignore
		// Но для большинства случаев достаточно
		indexFiles := make(map[string]bool)
		for _, entry := range indexEntries {
			indexFiles[entry.path] = true
		}

		// Полная проверка всех файлов может быть медленной
		err = filepath.Walk(repoRoot, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr == nil && info != nil {
				if info.IsDir() && info.Name() == ".git" {
					return filepath.SkipDir
				}

				if info.IsDir() {
					return nil
				}

				if relPath, relErr := filepath.Rel(repoRoot, path); relErr == nil {
					relPath = filepath.ToSlash(relPath)

					// Если файл не в индексе и не игнорируется, репозиторий dirty
					if !indexFiles[relPath] {
						if !strings.HasPrefix(relPath, ".git/") && !strings.HasPrefix(relPath, ".git") {
							// Это новый файл - репозиторий dirty
							return errNewFileFound
						}
					}
				}
			}
			return nil
		})

		if errors.Is(err, errNewFileFound) {
			isDirty = true
			err = nil
			return
		}
	}

	return
}

type indexEntry struct {
	path  string
	size  int64
	mtime int64
}

func parseGitIndex(indexPath string) (entries []indexEntry, err error) {

	var data []byte
	if data, err = os.ReadFile(indexPath); err != nil {
		return
	}

	if len(data) < 12 {
		err = errors.New("index file too short")
		return
	}

	if string(data[0:4]) != "DIRC" {
		err = errors.New("invalid index signature")
		return
	}

	// Читаем количество записей (4 bytes после signature)
	entryCount := int(data[8])<<24 | int(data[9])<<16 | int(data[10])<<8 | int(data[11])

	entries = make([]indexEntry, 0)
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
		// - sha (20 bytes)
		// - flags (2 bytes)
		// - path (variable, null-terminated, padded to multiple of 8)

		// mtime хранится как секунды (первые 4 байта) + наносекунды (последние 4 байта)
		// Для сравнения используем только секунды, так как os.FileInfo.ModTime() возвращает время с точностью до секунды
		mtimeSeconds := int64(data[offset+8])<<24 | int64(data[offset+9])<<16 | int64(data[offset+10])<<8 | int64(data[offset+11])

		size := int64(data[offset+36])<<24 | int64(data[offset+37])<<16 | int64(data[offset+38])<<8 | int64(data[offset+39])

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

		entries = append(entries, indexEntry{
			path:  path,
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

func getGitUser(gitDir string) (user string, email string, err error) {

	configPath := filepath.Join(gitDir, "config")
	var file *os.File
	if file, err = os.Open(configPath); err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if after, ok := strings.CutPrefix(line, "name = "); ok {
			user = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(line, "email = "); ok {
			email = strings.TrimSpace(after)
		}
	}

	if err = scanner.Err(); err != nil {
		return
	}

	return
}

func getGitRemoteURL(gitDir string) (url string, err error) {

	configPath := filepath.Join(gitDir, "config")
	var file *os.File
	if file, err = os.Open(configPath); err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	inOriginSection := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Может быть в формате [remote "origin"] или [remote.origin]
		if strings.HasPrefix(line, "[remote \"origin\"]") || strings.HasPrefix(line, "[remote.origin]") {
			inOriginSection = true
			continue
		}

		// Если встретили другую секцию, выходим
		if strings.HasPrefix(line, "[") {
			if inOriginSection {
				break
			}
			inOriginSection = false
			continue
		}

		// Может быть в формате "url = ..." или "	url = ..." (с табуляцией)
		if inOriginSection {
			lineTrimmed := strings.TrimSpace(line)
			if after, ok := strings.CutPrefix(lineTrimmed, "url = "); ok {
				url = strings.TrimSpace(after)
				return
			}
		}
	}

	if err = scanner.Err(); err != nil {
		return
	}

	return
}
