// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package cache

import (
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-json"

	"tgp/core/i18n"
	"tgp/internal/merkle"
	"tgp/internal/model"
	"tgp/plugins/astg/marker"
)

const (
	BaseDir = "/tg/astg/cache"
)

func GetProject(rootDir string) (project *model.Project, fromCache bool, projectID string) {

	var err error
	if projectID, err = GetProjectID(rootDir); err != nil {
		slog.Debug(i18n.Msg("cannot compute project ID, skipping cache"), slog.Any("error", err))
		return
	}

	branch := getGitBranchForCache(rootDir)
	cacheFile := GetCachePath(projectID, branch)

	project, fromCache = loadEntry(rootDir, cacheFile, projectID)
	if fromCache {
		slog.Debug(i18n.Msg("using cached project"), slog.String("cacheFile", cacheFile))
	}

	return
}

func SaveProject(projectID string, project *model.Project, rootDir string, contractsDir string, excludeDirs []string) {

	project.ProjectID = projectID

	var err error
	var paths []string
	if paths, err = marker.DiscoverPaths(rootDir, contractsDir, excludeDirs); err != nil {
		slog.Debug(i18n.Msg("failed to discover paths, skipping cache"), slog.Any("error", err))
		return
	}

	var files map[string]string
	if files, err = merkle.FileHashes(rootDir, paths); err != nil {
		slog.Debug(i18n.Msg("failed to compute file hashes, skipping cache"), slog.Any("error", err))
		return
	}

	entry := cacheEntry{Project: project, Files: files}
	branch := ""
	if project.Git != nil && project.Git.Branch != "" {
		branch = project.Git.Branch
	}
	cacheFile := GetCachePath(projectID, branch)

	if saveErr := saveEntry(cacheFile, &entry); saveErr != nil {
		slog.Debug(i18n.Msg("failed to save cache"), slog.Any("error", saveErr), slog.String("cacheFile", cacheFile))
	} else {
		slog.Debug(i18n.Msg("project cached"), slog.String("cacheFile", cacheFile))
	}
}

func getGitBranchForCache(rootDir string) (branch string) {

	var gitDir string
	var err error
	dir := rootDir
	for {
		gitPath := filepath.Join(dir, ".git")
		var info os.FileInfo
		if info, err = os.Stat(gitPath); err == nil {
			if info.IsDir() {
				gitDir = gitPath
				break
			}
			var content []byte
			if content, err = os.ReadFile(gitPath); err == nil {
				gitDir = strings.TrimSpace(string(content))
				if strings.HasPrefix(gitDir, "gitdir: ") {
					gitDir = strings.TrimPrefix(gitDir, "gitdir: ")
					if !filepath.IsAbs(gitDir) {
						gitDir = filepath.Join(dir, gitDir)
					}
					break
				}
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}

	if gitDir == "" {
		return
	}

	var headContent []byte
	if headContent, err = readGitFile(gitDir, "HEAD"); err != nil {
		return
	}

	headStr := strings.TrimSpace(string(headContent))

	if strings.HasPrefix(headStr, "ref: ") {
		refPath := strings.TrimPrefix(headStr, "ref: ")
		refPath = strings.TrimSpace(refPath)
		if strings.HasPrefix(refPath, "refs/heads/") {
			branch = strings.TrimPrefix(refPath, "refs/heads/")
			return branch
		}
	}

	return
}

func readGitFile(gitDir string, fileName string) (content []byte, err error) {

	var baseDir string
	if baseDir, err = filepath.Abs(filepath.Clean(gitDir)); err != nil {
		return
	}
	targetPath := filepath.Join(baseDir, fileName)
	var targetAbsPath string
	if targetAbsPath, err = filepath.Abs(filepath.Clean(targetPath)); err != nil {
		return
	}
	if !strings.HasPrefix(targetAbsPath, baseDir+string(os.PathSeparator)) {
		return nil, fmt.Errorf("unsafe git path: %s", targetAbsPath)
	}
	return os.ReadFile(targetAbsPath)
}

func loadEntry(rootDir string, cacheFile string, projectID string) (project *model.Project, valid bool) {

	var err error
	var info os.FileInfo
	if info, err = os.Stat(cacheFile); err != nil {
		if os.IsNotExist(err) {
			slog.Debug(i18n.Msg("cache file not found"), slog.String("cacheFile", cacheFile))
		} else {
			slog.Debug(i18n.Msg("failed to stat cache file"), slog.Any("error", err), slog.String("cacheFile", cacheFile))
		}
		return
	}

	if info.IsDir() {
		slog.Debug(i18n.Msg("cache path is directory, not a file"), slog.String("cacheFile", cacheFile))
		return
	}

	var file *os.File
	if file, err = os.Open(cacheFile); err != nil {
		slog.Debug(i18n.Msg("failed to open cache file"), slog.Any("error", err), slog.String("cacheFile", cacheFile))
		return
	}
	defer file.Close()

	var gzipReader *gzip.Reader
	if gzipReader, err = gzip.NewReader(file); err != nil {
		slog.Debug(i18n.Msg("failed to create gzip reader"), slog.Any("error", err), slog.String("cacheFile", cacheFile))
		_ = os.Remove(cacheFile)
		return
	}
	defer gzipReader.Close()

	var jsonData []byte
	if jsonData, err = io.ReadAll(gzipReader); err != nil {
		slog.Debug(i18n.Msg("failed to read cache data"), slog.Any("error", err), slog.String("cacheFile", cacheFile))
		_ = os.Remove(cacheFile)
		return
	}

	var entry cacheEntry
	if err = json.Unmarshal(jsonData, &entry); err != nil {
		slog.Debug(i18n.Msg("failed to unmarshal cache"), slog.Any("error", err), slog.String("cacheFile", cacheFile))
		_ = os.Remove(cacheFile)
		return
	}

	if entry.Project == nil || entry.Project.ProjectID == "" {
		slog.Debug(i18n.Msg("cached project has no ProjectID"), slog.String("cacheFile", cacheFile))
		return
	}

	if entry.Project.ProjectID != projectID {
		slog.Debug(i18n.Msg("project ID mismatch"),
			slog.String("expected", projectID),
			slog.String("got", entry.Project.ProjectID),
			slog.String("cacheFile", cacheFile))
		return
	}

	if len(entry.Files) == 0 {
		slog.Debug(i18n.Msg("cached entry has no Files"), slog.String("cacheFile", cacheFile))
		return
	}

	paths := make([]string, 0, len(entry.Files))
	for p := range entry.Files {
		paths = append(paths, p)
	}
	var current map[string]string
	if current, err = merkle.FileHashes(rootDir, paths); err != nil {
		slog.Debug(i18n.Msg("failed to compute current file hashes"), slog.Any("error", err))
		return
	}

	for path, want := range entry.Files {
		if current[path] != want {
			slog.Debug(i18n.Msg("file hash mismatch"), slog.String("path", path))
			return
		}
	}

	return entry.Project, true
}

func saveEntry(cacheFile string, entry *cacheEntry) (err error) {

	cacheDir := filepath.Dir(cacheFile)
	if err = os.MkdirAll(cacheDir, 0755); err != nil {
		slog.Debug(i18n.Msg("failed to create cache directory"), slog.Any("error", err), slog.String("cacheDir", cacheDir))
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	var jsonData []byte
	if jsonData, err = json.MarshalIndent(entry, "", "  "); err != nil {
		slog.Debug(i18n.Msg("failed to marshal cache entry"), slog.Any("error", err))
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	var file *os.File
	if file, err = os.Create(cacheFile); err != nil {
		slog.Debug(i18n.Msg("failed to create cache file"), slog.Any("error", err), slog.String("cacheFile", cacheFile))
		return fmt.Errorf("failed to create cache file: %w", err)
	}

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()
	if _, err = gzipWriter.Write(jsonData); err != nil {
		file.Close()
		_ = os.Remove(cacheFile)
		slog.Debug(i18n.Msg("failed to write compressed data"), slog.Any("error", err))
		return fmt.Errorf("failed to write compressed data: %w", err)
	}

	if err = gzipWriter.Close(); err != nil {
		file.Close()
		_ = os.Remove(cacheFile)
		slog.Debug(i18n.Msg("failed to close gzip writer"), slog.Any("error", err))
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	if err = file.Close(); err != nil {
		_ = os.Remove(cacheFile)
		slog.Debug(i18n.Msg("failed to close cache file"), slog.Any("error", err))
		return fmt.Errorf("failed to close cache file: %w", err)
	}

	return
}

func GetCachePath(projectID string, branch string) (s string) {

	normalizedBranch := marker.NormalizeBranchName(branch)
	cacheFile := fmt.Sprintf("%s/%s/%s.astg", BaseDir, projectID, normalizedBranch)
	return cacheFile
}
