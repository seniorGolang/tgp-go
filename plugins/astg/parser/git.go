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
			err = errors.New("git repository not found")
			return
		}
		dir = parent
	}
}

func getGitCommit(gitDir string) (commit string, err error) {

	// Читаем HEAD
	headPath := filepath.Join(gitDir, "HEAD")
	var headContent []byte
	if headContent, err = os.ReadFile(headPath); err != nil {
		return
	}

	headStr := strings.TrimSpace(string(headContent))

	// Если HEAD указывает на ветку (ref: refs/heads/master)
	if strings.HasPrefix(headStr, "ref: ") {
		refPath := strings.TrimPrefix(headStr, "ref: ")
		refPath = strings.TrimSpace(refPath)
		commitPath := filepath.Join(gitDir, refPath)
		var commitBytes []byte
		if commitBytes, err = os.ReadFile(commitPath); err != nil {
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
	if strings.HasPrefix(headStr, "ref: ") {
		refPath := strings.TrimPrefix(headStr, "ref: ")
		refPath = strings.TrimSpace(refPath)
		if strings.HasPrefix(refPath, "refs/heads/") {
			branch = strings.TrimPrefix(refPath, "refs/heads/")
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

		tagPath := filepath.Join(tagsDir, entry.Name())
		var tagContent []byte
		if tagContent, err = os.ReadFile(tagPath); err != nil {
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

		tagObjPath := filepath.Join(gitDir, "objects", tagHash[:2], tagHash[2:])
		var tagObjContent []byte
		if tagObjContent, err = os.ReadFile(tagObjPath); err == nil {
			// Парсим объект тега (формат: object <hash>\ntype tag\n...)
			// Ищем строку "object " и извлекаем хеш коммита
			content := string(tagObjContent)
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "object ") {
					objHash := strings.TrimSpace(strings.TrimPrefix(line, "object "))
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

func isGitDirty(gitDir string, repoRoot string) (isDirty bool, err error) {

	indexPath := filepath.Join(gitDir, "index")
	var statErr error
	if _, statErr = os.Stat(indexPath); statErr != nil {
		return
	}

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
		if relPath, walkErr = filepath.Rel(repoRoot, path); walkErr != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		// Если файл не в индексе и не игнорируется, репозиторий dirty
		if !indexFiles[relPath] {
			if !strings.HasPrefix(relPath, ".git/") && !strings.HasPrefix(relPath, ".git") {
				// Это новый файл - репозиторий dirty
				return errors.New("new file found")
			}
		}

		return nil
	})

	if err != nil && err.Error() == "new file found" {
		isDirty = true
		err = nil
		return
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
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "name = ") {
			user = strings.TrimSpace(strings.TrimPrefix(line, "name = "))
		} else if strings.HasPrefix(line, "email = ") {
			email = strings.TrimSpace(strings.TrimPrefix(line, "email = "))
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
	defer file.Close()

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

		// В секции origin ищем url
		// Может быть в формате "url = ..." или "	url = ..." (с табуляцией)
		if inOriginSection {
			lineTrimmed := strings.TrimSpace(line)
			if strings.HasPrefix(lineTrimmed, "url = ") {
				url = strings.TrimSpace(strings.TrimPrefix(lineTrimmed, "url = "))
				return
			}
		}
	}

	if err = scanner.Err(); err != nil {
		return
	}

	return
}
