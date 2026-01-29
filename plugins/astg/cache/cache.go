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
	"tgp/internal/model"
	"tgp/plugins/astg/marker"
)

const (
	// CacheBaseDir базовый путь к директории кэша.
	CacheBaseDir = "/tg/cache/astg"
)

func GetProject(rootDir string) (project *model.Project, fromCache bool, projectID string, currentMarker string) {

	var err error
	projectID, err = GetProjectID(rootDir)
	if err != nil {
		// Если не удалось вычислить ProjectID, пропускаем кэширование
		slog.Debug(i18n.Msg("cannot compute project ID, skipping cache"), slog.String("error", err.Error()))
		return nil, false, "", ""
	}

	branch := getGitBranchForCache(rootDir)
	normalizedBranch := NormalizeBranch(branch)

	// Путь к кэшу
	cacheFile := GetCachePath(projectID, normalizedBranch)

	// Вычисление текущего маркера
	if currentMarker, err = marker.ComputeMarker(rootDir); err != nil {
		// Если не удалось вычислить маркер, пропускаем кэширование
		slog.Debug(i18n.Msg("failed to compute marker, skipping cache"), slog.String("error", err.Error()))
		return nil, false, projectID, ""
	}

	// Загрузка и валидация кэша
	cachedProject, valid := loadProject(cacheFile, projectID, currentMarker)
	if valid && cachedProject != nil {
		// Кэш валиден - используем его
		slog.Debug(i18n.Msg("using cached project"), slog.String("cacheFile", cacheFile))
		return cachedProject, true, projectID, currentMarker
	}

	// Кэш невалиден или отсутствует
	return nil, false, projectID, currentMarker
}

func SaveProject(projectID string, currentMarker string, project *model.Project) {

	// Заполняем метаданные в проекте
	project.ProjectID = projectID
	project.Marker = currentMarker

	branch := ""
	if project.Git != nil && project.Git.Branch != "" {
		branch = project.Git.Branch
	}
	normalizedBranch := NormalizeBranch(branch)
	cacheFile := GetCachePath(projectID, normalizedBranch)

	// Сохранение в кэш (ошибки игнорируются, логируются на DEBUG)
	if saveErr := saveProject(cacheFile, project); saveErr != nil {
		slog.Debug(i18n.Msg("failed to save cache"), slog.String("error", saveErr.Error()), slog.String("cacheFile", cacheFile))
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
			return ""
		}
		dir = parent
	}

	if gitDir == "" {
		return ""
	}

	// Читаем HEAD для получения ветки
	headPath := filepath.Join(gitDir, "HEAD")
	var headContent []byte
	if headContent, err = os.ReadFile(headPath); err != nil {
		return ""
	}

	headStr := strings.TrimSpace(string(headContent))

	// Если HEAD указывает на ветку
	if strings.HasPrefix(headStr, "ref: ") {
		refPath := strings.TrimPrefix(headStr, "ref: ")
		refPath = strings.TrimSpace(refPath)
		if strings.HasPrefix(refPath, "refs/heads/") {
			branch = strings.TrimPrefix(refPath, "refs/heads/")
			return branch
		}
	}

	return ""
}

func loadProject(cacheFile string, projectID string, currentMarker string) (project *model.Project, valid bool) {

	var err error
	var info os.FileInfo
	if info, err = os.Stat(cacheFile); err != nil {
		if os.IsNotExist(err) {
			slog.Debug(i18n.Msg("cache file not found"), slog.String("cacheFile", cacheFile))
		} else {
			slog.Debug(i18n.Msg("failed to stat cache file"), slog.String("error", err.Error()), slog.String("cacheFile", cacheFile))
		}
		return
	}

	if info.IsDir() {
		slog.Debug(i18n.Msg("cache path is directory, not a file"), slog.String("cacheFile", cacheFile))
		return
	}

	var file *os.File
	if file, err = os.Open(cacheFile); err != nil {
		slog.Debug(i18n.Msg("failed to open cache file"), slog.String("error", err.Error()), slog.String("cacheFile", cacheFile))
		return
	}
	defer file.Close()

	var gzipReader *gzip.Reader
	if gzipReader, err = gzip.NewReader(file); err != nil {
		slog.Debug(i18n.Msg("failed to create gzip reader"), slog.String("error", err.Error()), slog.String("cacheFile", cacheFile))
		// Удалить поврежденный файл
		_ = os.Remove(cacheFile)
		return
	}
	defer gzipReader.Close()

	var jsonData []byte
	if jsonData, err = io.ReadAll(gzipReader); err != nil {
		slog.Debug(i18n.Msg("failed to read cache data"), slog.String("error", err.Error()), slog.String("cacheFile", cacheFile))
		_ = os.Remove(cacheFile)
		return
	}

	// 5. Десериализация JSON
	cachedProject := &model.Project{}
	if err = json.Unmarshal(jsonData, cachedProject); err != nil {
		slog.Debug(i18n.Msg("failed to unmarshal cache"), slog.String("error", err.Error()), slog.String("cacheFile", cacheFile))
		_ = os.Remove(cacheFile)
		return
	}

	// 6. Валидация ProjectID - должен полностью совпадать
	if cachedProject.ProjectID == "" {
		slog.Debug(i18n.Msg("cached project has no ProjectID"), slog.String("cacheFile", cacheFile))
		return
	}

	if cachedProject.ProjectID != projectID {
		slog.Debug(i18n.Msg("project ID mismatch"),
			slog.String("expected", projectID),
			slog.String("got", cachedProject.ProjectID),
			slog.String("cacheFile", cacheFile))
		// Кэш невалиден, но не удаляем файл - он может быть для другого проекта
		return
	}

	// 7. Валидация маркера - должен полностью совпадать
	if cachedProject.Marker == "" {
		slog.Debug(i18n.Msg("cached project has no Marker"), slog.String("cacheFile", cacheFile))
		return
	}

	if cachedProject.Marker != currentMarker {
		slog.Debug(i18n.Msg("marker mismatch"),
			slog.String("expected", currentMarker),
			slog.String("got", cachedProject.Marker),
			slog.String("cacheFile", cacheFile))
		// Маркер изменился - проект изменился, кэш невалиден
		return
	}

	// 8. Кэш валиден - ProjectID и Marker полностью совпадают
	return cachedProject, true
}

func saveProject(cacheFile string, project *model.Project) (err error) {

	// 1. Создание директории кэша
	cacheDir := filepath.Dir(cacheFile)
	if err = os.MkdirAll(cacheDir, 0755); err != nil {
		slog.Debug(i18n.Msg("failed to create cache directory"), slog.String("error", err.Error()), slog.String("cacheDir", cacheDir))
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	var jsonData []byte
	if jsonData, err = json.MarshalIndent(project, "", "  "); err != nil {
		slog.Debug(i18n.Msg("failed to marshal project"), slog.String("error", err.Error()))
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	var file *os.File
	if file, err = os.Create(cacheFile); err != nil {
		slog.Debug(i18n.Msg("failed to create cache file"), slog.String("error", err.Error()), slog.String("cacheFile", cacheFile))
		return fmt.Errorf("failed to create cache file: %w", err)
	}

	// 4. Сжатие JSON (gzip) и запись
	gzipWriter := gzip.NewWriter(file)
	if _, err = gzipWriter.Write(jsonData); err != nil {
		file.Close()
		os.Remove(cacheFile)
		slog.Debug(i18n.Msg("failed to write compressed data"), slog.String("error", err.Error()))
		return fmt.Errorf("failed to write compressed data: %w", err)
	}

	if err = gzipWriter.Close(); err != nil {
		file.Close()
		os.Remove(cacheFile)
		slog.Debug(i18n.Msg("failed to close gzip writer"), slog.String("error", err.Error()))
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	if err = file.Close(); err != nil {
		os.Remove(cacheFile)
		slog.Debug(i18n.Msg("failed to close cache file"), slog.String("error", err.Error()))
		return fmt.Errorf("failed to close cache file: %w", err)
	}

	return
}

func GetCachePath(projectID string, branch string) string {

	normalizedBranch := NormalizeBranch(branch)
	cacheFile := fmt.Sprintf("%s/%s/%s.astg", CacheBaseDir, projectID, normalizedBranch)
	return cacheFile
}
