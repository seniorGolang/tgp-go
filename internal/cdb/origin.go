package cdb

import (
	"net/url"
	"strings"
)

func NormalizeRemoteURLToHostPath(remoteURL string) (hostPath string) {

	remoteURL = strings.TrimLeft(strings.TrimSpace(remoteURL), ".")

	if strings.HasPrefix(remoteURL, "git@") {
		remoteURL = "https://" + strings.Replace(remoteURL[len("git@"):], ":", "/", 1)
	} else if strings.HasPrefix(remoteURL, "ssh://") {
		remoteURL = "https://" + strings.TrimPrefix(remoteURL[len("ssh://"):], "git@")
	}

	var err error
	var parsed *url.URL
	if parsed, err = url.Parse(remoteURL); err != nil || parsed.Host == "" {
		return ""
	}

	path := strings.TrimSuffix(strings.TrimPrefix(parsed.Path, "/"), ".git")
	return strings.TrimSuffix(parsed.Hostname()+"/"+path, "/")
}

func RemoteURLToProjectKey(remoteURL string) (projectKey string) {
	return sanitizeProjectKey(strings.ReplaceAll(NormalizeRemoteURLToHostPath(remoteURL), "/", "."))
}

func sanitizeProjectKey(projectKey string) (safe string) {

	safe = strings.TrimLeft(projectKey, ".")
	safe = strings.ReplaceAll(safe, "..", "-")
	safe = strings.ReplaceAll(safe, ":", ".")
	return
}

// ModulePathToProjectKey превращает путь модуля Go в ключ проекта (слэши в точки).
func ModulePathToProjectKey(modulePath string) (projectKey string) {
	return strings.ReplaceAll(strings.Trim(modulePath, "/"), "/", ".")
}

// Санитизация и префикс "local." для односегментных ключей.
func ProjectKeyForStorage(projectKey string) (storageKey string) {

	safe := sanitizeProjectKey(projectKey)

	if safe == "" || strings.Contains(safe, ".") {
		return safe
	}
	return "local." + safe
}
