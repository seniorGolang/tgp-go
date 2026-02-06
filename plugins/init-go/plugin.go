package main

import (
	_ "embed"
	"errors"
	"fmt"
	"path"
	"path/filepath"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/core/plugin"
	"tgp/plugins/init-go/generator"
)

//go:embed plugin.md
var docContent string

// BaseGoPlugin реализует интерфейсы Plugin и InitGenerator.
type BaseGoPlugin struct{}

// Execute выполняет основную логику плагина: генерирует базовый Go-проект по параметрам из request.
func (p *BaseGoPlugin) Execute(_ string, request data.Storage, _ ...string) (response data.Storage, err error) {

	response = request
	moduleName, err := data.Get[string](request, "module")
	if err != nil || moduleName == "" {
		return nil, errors.New(i18n.Msg("module option is required"))
	}
	var out, rest, jsonRPC string
	out, _ = data.Get[string](request, "out")
	rest, _ = data.Get[string](request, "rest")
	jsonRPC, _ = data.Get[string](request, "json-rpc")
	out = path.Join("/", out)
	if err = generator.Generate(out, moduleName, jsonRPC, rest); err != nil {
		absOut, _ := filepath.Abs(out)
		err = fmt.Errorf("init go in %s %s: %w", absOut, i18n.Msg("init generate"), err)
		return
	}
	return
}

// Info возвращает информацию о плагине.
func (p *BaseGoPlugin) Info() (info plugin.Info, err error) {

	info = plugin.Info{
		Name:             "init-go",
		Description:      i18n.Msg("Base Go project generator"),
		Author:           "AlexK <seniorGolang@gmail.com>",
		License:          "MIT",
		Category:         "utility",
		Doc:              docContent,
		AllowedShellCMDs: []string{"go"},
		AllowedStdOut:    true,
		AllowedStdErr:    true,
		AllowedPaths:     map[string]string{"@root": "w"},
		AllowedEnvVars:   []string{"PATH", "GOROOT", "GOPATH"},
		Commands: []plugin.Command{
			{
				Path:        []string{"init", "go"},
				Description: i18n.Msg("Initialize base Go project with contracts and service skeleton"),
				Options: []plugin.Option{
					{Name: "module", Short: "m", Type: "string", Description: i18n.Msg("Module name"), Required: true},
					{Name: "out", Short: "o", Type: "string", Description: i18n.Msg("Output directory (default: current)"), Required: false},
					{Name: "json-rpc", Type: "string", Description: i18n.Msg("JSON-RPC interface names, comma-separated (e.g. some,demo)"), Required: false},
					{Name: "rest", Type: "string", Description: i18n.Msg("REST interface names, comma-separated (e.g. example,siteNova)"), Required: false},
				},
			},
		},
	}
	return
}

// Generate создаёт базовый Go-проект. rootDir — корень, out — относительный путь от корня; плагин разрешает путь и передаёт в генератор.
func (p *BaseGoPlugin) Generate(rootDir string, out string, moduleName string, jsonRPC string, rest string) (err error) {

	outDir := filepath.Join(rootDir, out)
	if err = generator.Generate(outDir, moduleName, jsonRPC, rest); err != nil {
		return fmt.Errorf("%s: %w", i18n.Msg("init generate"), err)
	}
	return
}
