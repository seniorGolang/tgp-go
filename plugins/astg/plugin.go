package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-json"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/core/plugin"
	"tgp/internal"
	"tgp/internal/helper"
	"tgp/internal/model"
	"tgp/plugins/astg/cache"
	"tgp/plugins/astg/generator"
	"tgp/plugins/astg/marker"
	"tgp/plugins/astg/parser"
)

//go:embed plugin.md
var pluginDoc string

// AstgPlugin реализует интерфейс Plugin.
type AstgPlugin struct{}

// Execute выполняет основную логику плагина.
func (p *AstgPlugin) Execute(rootDir string, request data.Storage, path ...string) (response data.Storage, err error) {

	// Настраиваем дефолтный slog с контекстом плагина
	slog.SetDefault(slog.Default().With(
		slog.String("plugin", "astg"),
	))

	// Создаем Response и копируем все данные из request
	response = data.NewStorage()
	if request != nil {
		if storageMap, ok := request.(*data.MapStorage); ok {
			// Копируем map напрямую
			responseMap := response.(*data.MapStorage)
			*responseMap = make(data.MapStorage, len(*storageMap))
			for k, raw := range *storageMap {
				(*responseMap)[k] = raw
			}
		}
	}

	// Если project уже есть в request, не пересоздаем его
	if request != nil && request.Has("project") {
		slog.Debug(i18n.Msg("project already exists in request, skipping analysis"))
		return
	}

	// Получаем contracts из request или используем значение по умолчанию
	contractsDir := "contracts"
	var contractsStr string
	if contractsStr, err = data.Get[string](request, "contracts"); err == nil && contractsStr != "" {
		contractsDir = contractsStr
	}

	// Получаем список интерфейсов для фильтрации
	var ifaces []string
	var ifacesStr string
	if ifacesStr, err = data.Get[string](request, "ifaces"); err == nil && ifacesStr != "" {
		parts := strings.FieldsFunc(ifacesStr, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t'
		})
		ifaces = make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				ifaces = append(ifaces, part)
			}
		}
	}

	// Определяем значение для вывода ifaces
	ifacesDisplay := "all"
	if len(ifaces) > 0 {
		ifacesDisplay = strings.Join(ifaces, ", ")
	}

	// Получаем опцию no-cache
	noCache, _ := data.Get[bool](request, "no-cache")

	slog.Info(i18n.Msg("analyzing project"),
		slog.String("contractsDir", contractsDir),
		slog.String("ifaces", ifacesDisplay),
		slog.String("version", internal.Version),
		slog.Bool("no-cache", noCache),
	)

	// Пытаемся получить проект из кэша (если опция no-cache не установлена)
	// GetProject возвращает также вычисленные projectID и marker для последующего использования
	var fromCache bool
	var projectID string
	var currentMarker string
	var project *model.Project

	if noCache {
		// Если no-cache установлен, игнорируем кэш и всегда выполняем парсинг
		fromCache = false
		// Вычисляем projectID и marker заранее для последующего сохранения в кэш
		if projectID, err = cache.GetProjectID(rootDir); err != nil {
			slog.Debug(i18n.Msg("failed to compute project ID"), slog.String("error", err.Error()))
		}
		if currentMarker, err = marker.ComputeMarker(rootDir); err != nil {
			slog.Debug(i18n.Msg("failed to compute marker"), slog.String("error", err.Error()))
		}
	} else {
		project, fromCache, projectID, currentMarker = cache.GetProject(internal.ProjectRoot)
	}

	// Если проект не из кэша, выполняем парсинг и сохраняем в кэш
	if !fromCache {
		// Выполняем парсинг проекта (всегда собираем все контракты, игнорируя ifaces)
		if project, err = parser.CollectWithExcludeDirs(internal.Version, contractsDir, nil); err != nil {
			err = fmt.Errorf("%s: %w", i18n.Msg("failed to collect project"), err)
			response = nil
			return
		}

		// Если projectID и marker уже вычислены в GetProject, используем их
		// Если нет, вычисляем заново
		if projectID == "" {
			if projectID, err = cache.GetProjectID(rootDir); err == nil {
				project.ProjectID = projectID
			}
		} else {
			project.ProjectID = projectID
		}

		if currentMarker == "" {
			if currentMarker, err = marker.ComputeMarker(rootDir); err == nil {
				project.Marker = currentMarker
			}
		} else {
			project.Marker = currentMarker
		}

		// Сохраняем проект в кэш (используем уже вычисленные projectID и marker)
		// В кэш сохраняются полные данные без фильтрации
		if projectID != "" {
			cache.SaveProject(projectID, currentMarker, project)
		}
	}

	// Применяем фильтрацию по ifaces после загрузки проекта (из кэша или после парсинга)
	// Фильтрация применяется к копии проекта, чтобы не изменять данные в кэше
	if len(ifaces) > 0 {
		var filteredContracts []*model.Contract
		if filteredContracts, err = helper.FilterContractsByInterfaces(project, ifaces); err != nil {
			err = fmt.Errorf("%s: %w", i18n.Msg("failed to filter contracts by interfaces"), err)
			response = nil
			return
		}
		// Создаем копию проекта с отфильтрованными контрактами
		filteredProject := *project
		filteredProject.Contracts = filteredContracts
		project = &filteredProject
	}

	// Выводим сводку по каждому контракту отдельно
	for _, contract := range project.Contracts {
		// Подсчитываем уникальные ошибки во всех реализациях
		errorTypesMap := make(map[string]bool)
		for _, impl := range contract.Implementations {
			for _, method := range impl.MethodsMap {
				for _, errorType := range method.ErrorTypes {
					errorTypesMap[errorType.FullName] = true
				}
			}
		}

		slog.Info(contract.Name,
			slog.String("pkgPath", contract.PkgPath),
			slog.Int("methodsCount", len(contract.Methods)),
			slog.Int("implementationsCount", len(contract.Implementations)),
			slog.Int("errorsCount", len(errorTypesMap)),
		)
	}

	// Сохраняем project в файл .tg/project.json только в DEBUG режиме
	// В WASM окружении корень файловой системы - это корень проекта
	if slog.Default().Handler().Enabled(context.Background(), slog.LevelDebug) {
		projectPath := "/.tg/project.json"

		slog.Debug(i18n.Msg("saving project"), slog.String("rootDir", rootDir), slog.String("projectPath", projectPath))

		if err = os.MkdirAll(filepath.Dir(projectPath), 0755); err == nil {
			var projectJSON []byte
			if projectJSON, err = json.MarshalIndent(project, "", "  "); err == nil {
				if err = os.WriteFile(projectPath, projectJSON, 0600); err == nil {
					slog.Debug(i18n.Msg("project saved"), slog.String("path", projectPath), slog.Int("typesCount", len(project.Types)))
				} else {
					slog.Warn(i18n.Msg("failed to save project"), slog.String("error", err.Error()), slog.String("path", projectPath))
				}
			} else {
				slog.Warn(i18n.Msg("failed to marshal project"), slog.String("error", err.Error()))
			}
		} else {
			slog.Warn(i18n.Msg("failed to create project directory"), slog.String("error", err.Error()), slog.String("path", filepath.Dir(projectPath)))
		}
	}

	// Добавляем project в response
	if err = response.Set("project", project); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("failed to set project in response"), err)
		response = nil
		return
	}

	slog.Info(i18n.Msg("analysis completed"))
	return
}

// Info возвращает информацию о плагине.
func (p *AstgPlugin) Info() (info plugin.Info, err error) {

	info = plugin.Info{
		Name:        "astg",
		Doc:         pluginDoc,
		Description: i18n.Msg("Project AST parser plugin"),
		Author:      "AlexK <seniorGolang@gmail.com>",
		License:     "MIT",
		Category:    "parser",
		AllowedEnvVars: []string{
			"GOPATH",     // Для поиска пакетов в GOPATH/src и модулей в GOPATH/pkg/mod
			"GOROOT",     // Для поиска стандартной библиотеки
			"GOMODCACHE", // Для поиска модулей в кэше модулей
			"GOOS",       // Для build.Context при парсинге файлов с build tags
			"GOARCH",     // Для build.Context при парсинге файлов с build tags
		},
		InitPkgs: []string{"astg"},
		AllowedPaths: map[string]string{
			"@go":            "w", // Доступ к директории с go.mod (монтируется хостом в корень "/")
			"$GOPATH":        "r", // Для чтения пакетов из GOPATH/src (для go/types и go/build)
			"$GOROOT":        "r", // Для чтения стандартной библиотеки Go (для go/types и go/build)
			"$GOMODCACHE":    "r", // Для чтения модулей из кэша (для go/types и go/build)
			"@tg/cache/astg": "w", // Доступ на запись к кэшу проектов
		},
	}
	return
}

// Generate генерирует пакет astg с автономными типами.
// Реализует интерфейс InitGenerator для генерации кода при tg plugin init.
func (p *AstgPlugin) Generate(rootDir string, moduleName string) (err error) {

	slog.Info(i18n.Msg("generating astg package"), slog.String("rootDir", rootDir), slog.String("moduleName", moduleName))
	if err = generator.Generate(rootDir, moduleName); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("failed to generate astg package"), err)
		return
	}
	slog.Info(i18n.Msg("astg package generated successfully"))
	return
}

// Cleanup удаляет сгенерированные файлы.
// Реализует интерфейс InitGenerator для очистки при tg plugin upgrade.
func (p *AstgPlugin) Cleanup(rootDir string) (err error) {

	slog.Info(i18n.Msg("cleaning up astg package"), slog.String("rootDir", rootDir))
	if err = generator.Cleanup(rootDir); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("failed to cleanup astg package"), err)
		return
	}
	slog.Info(i18n.Msg("astg package cleaned up successfully"))
	return
}
