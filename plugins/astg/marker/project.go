// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package marker

import (
	"bufio"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/mod/modfile"
)

const (
	// base58Alphabet алфавит для Base58 кодирования.
	base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
)

func ProjectID(rootDir string) (id string, err error) {

	var gitDir string
	if gitDir, err = findGitDir(rootDir); err != nil {
		return "", fmt.Errorf("git repository not found: %w", err)
	}

	var remoteURL string
	remoteURL, _ = getGitRemoteURLFromConfig(gitDir)

	var normalizedRemote string
	if remoteURL != "" {
		normalizedRemote = normalizeGitRemoteURL(remoteURL)
	}

	var modulePath string
	if modulePath, err = getModulePath(rootDir); err != nil {
		return "", fmt.Errorf("go.mod not found or invalid: %w", err)
	}
	if modulePath == "" {
		return "", fmt.Errorf("module path is empty")
	}

	// 5. Сгенерировать UUIDv5
	// Namespace: стандартный DNS namespace UUID
	nameSpace := uuid.NameSpaceDNS
	var name string
	if normalizedRemote != "" {
		name = normalizedRemote + ":" + modulePath
	} else {
		name = modulePath
	}
	projectUUID := uuid.NewSHA1(nameSpace, []byte(name))

	// 6. Закодировать UUID в Base58
	id = encodeBase58(projectUUID[:])

	return id, nil
}

func getGitRemoteURLFromConfig(gitDir string) (remoteURL string, err error) {

	// Читаем из config файла напрямую
	configPath := filepath.Join(gitDir, "config")
	var file *os.File
	if file, err = os.Open(configPath); err != nil {
		// Если не удалось открыть config, возвращаем пустую строку без ошибки
		return "", nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inOriginSection := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

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
		if inOriginSection {
			lineTrimmed := strings.TrimSpace(line)
			if strings.HasPrefix(lineTrimmed, "url = ") {
				remoteURL = strings.TrimSpace(strings.TrimPrefix(lineTrimmed, "url = "))
				return remoteURL, nil
			}
		}
	}

	if err = scanner.Err(); err != nil {
		// Если ошибка чтения, возвращаем пустую строку без ошибки
		return "", nil
	}

	// Если remote URL не найден, возвращаем пустую строку без ошибки
	return "", nil
}

func normalizeGitRemoteURL(remoteURL string) (normalized string) {

	// Убираем пробелы
	remoteURL = strings.TrimSpace(remoteURL)

	// Обработка различных форматов URL
	// git@github.com:user/repo.git -> https://github.com/user/repo.git
	if strings.HasPrefix(remoteURL, "git@") {
		remoteURL = strings.Replace(remoteURL, "git@", "https://", 1)
		remoteURL = strings.Replace(remoteURL, ":", "/", 1)
	}

	// ssh://git@github.com/user/repo.git -> https://github.com/user/repo.git
	if strings.HasPrefix(remoteURL, "ssh://") {
		remoteURL = strings.Replace(remoteURL, "ssh://", "https://", 1)
		remoteURL = strings.Replace(remoteURL, "git@", "", 1)
		remoteURL = strings.Replace(remoteURL, ":", "/", 1)
	}

	// Парсим URL
	parsedURL, err := url.Parse(remoteURL)
	if err != nil {
		// Если не удалось распарсить, пытаемся обработать вручную
		return normalizeGitRemoteURLManual(remoteURL)
	}

	host := parsedURL.Host
	path := parsedURL.Path

	// Убираем порт, если стандартный
	host = strings.Replace(host, ":443", "", 1)
	host = strings.Replace(host, ":22", "", 1)

	// Убираем .git суффикс
	path = strings.TrimSuffix(path, ".git")
	path = strings.TrimPrefix(path, "/")

	// Объединяем host и path
	normalized = host + "/" + path
	normalized = strings.TrimSuffix(normalized, "/")

	return normalized
}

func normalizeGitRemoteURLManual(remoteURL string) (normalized string) {

	// Убираем протоколы
	remoteURL = strings.TrimPrefix(remoteURL, "https://")
	remoteURL = strings.TrimPrefix(remoteURL, "http://")
	remoteURL = strings.TrimPrefix(remoteURL, "git@")
	remoteURL = strings.TrimPrefix(remoteURL, "ssh://")

	remoteURL = strings.Replace(remoteURL, ":", "/", 1)

	// Убираем .git
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	// Убираем порты
	re := regexp.MustCompile(`:443(/|$)`)
	remoteURL = re.ReplaceAllString(remoteURL, "$1")
	re = regexp.MustCompile(`:22(/|$)`)
	remoteURL = re.ReplaceAllString(remoteURL, "$1")

	remoteURL = strings.Trim(remoteURL, "/")

	return remoteURL
}

func getModulePath(rootDir string) (modulePath string, err error) {

	var goModPath string
	currentDir := rootDir
	for {
		goModPath = filepath.Join(currentDir, "go.mod")
		var statErr error
		if _, statErr = os.Stat(goModPath); statErr == nil {
			break
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", fmt.Errorf("go.mod not found")
		}
		currentDir = parentDir
	}

	var modBytes []byte
	if modBytes, err = os.ReadFile(goModPath); err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	var modFile *modfile.File
	if modFile, err = modfile.Parse(goModPath, modBytes, nil); err != nil {
		return "", fmt.Errorf("failed to parse go.mod: %w", err)
	}

	if modFile.Module == nil {
		return "", fmt.Errorf("module declaration not found in go.mod")
	}

	modulePath = modFile.Module.Mod.Path
	return modulePath, nil
}

func encodeBase58(data []byte) (encoded string) {

	if len(data) == 0 {
		return ""
	}

	num := make([]byte, len(data))
	copy(num, data)

	var result []byte
	base := big.NewInt(58)
	zero := big.NewInt(0)
	bigNum := new(big.Int).SetBytes(num)

	for bigNum.Cmp(zero) > 0 {
		mod := new(big.Int)
		bigNum.DivMod(bigNum, base, mod)
		result = append(result, base58Alphabet[mod.Int64()])
	}

	for i := 0; i < len(data) && data[i] == 0; i++ {
		result = append(result, base58Alphabet[0])
	}

	// Разворачиваем результат
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}
