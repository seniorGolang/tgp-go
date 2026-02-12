// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"tgp/internal/common"
	"tgp/internal/markdown"

	"tgp/internal/model"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

type DocOptions struct {
	Enabled  bool   // Включена ли генерация документации (по умолчанию true)
	FilePath string // Полный путь к файлу документации (пусто = outDir/readme.md)
}

func (r *ClientRenderer) RenderReadmeGo(docOpts any) error {
	var err error
	outDir := r.outDir

	opts := DocOptions{Enabled: true}
	if docOpts != nil {
		if d, ok := docOpts.(DocOptions); ok {
			opts = d
		}
	}

	var buf bytes.Buffer
	md := markdown.NewMarkdown(&buf)

	md.H1("API Документация")
	md.PlainText("Автоматически сгенерированная документация API для Go клиента.")

	contracts := make([]*model.Contract, len(r.project.Contracts))
	copy(contracts, r.project.Contracts)
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].Name < contracts[j].Name
	})

	type tocItem struct {
		title  string
		level  int
		anchor string
	}
	tocItems := make([]tocItem, 0)

	hasJsonRPC := false

	for _, contract := range contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			hasJsonRPC = true
		}

		contractAnchor := contractAnchorID(contract.Name)
		_ = append(tocItems, tocItem{
			title:  contract.Name,
			level:  2,
			anchor: contractAnchor,
		})

		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			for _, method := range contract.Methods {
				if r.methodIsJsonRPC(contract, method) {
					methodAnchor := methodAnchorID(contract.Name, method.Name)
					_ = append(tocItems, tocItem{
						title:  method.Name,
						level:  3,
						anchor: methodAnchor,
					})
				}
			}
		}

		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
			for _, method := range contract.Methods {
				if r.methodIsHTTP(contract, method) {
					httpMethod := model.GetHTTPMethod(r.project, contract, method)
					httpPath := model.GetAnnotationValue(r.project, contract, method, nil, model.TagHttpPath, "/"+ToLowerCamel(method.Name))
					methodTitle := fmt.Sprintf("%s %s", httpMethod, httpPath)
					methodAnchor := methodAnchorID(contract.Name, methodTitle)
					_ = append(tocItems, tocItem{
						title:  methodTitle,
						level:  3,
						anchor: methodAnchor,
					})
				}
			}
		}
	}

	typeUsages := r.collectStructTypes()
	allTypes := make(map[string]*typeUsage)
	for key, usage := range common.SortedPairs(typeUsages) {
		allTypes[key] = usage
	}

	r.typeAnchorsSet = make(map[string]bool)
	for _, usage := range allTypes {
		r.typeAnchorsSet[typeAnchorID(usage.fullTypeName)] = true
	}

	typeKeys := common.SortedKeys(allTypes)
	sort.Strings(typeKeys)

	md.H2("Оглавление")
	md.PlainText(fmt.Sprintf("- [Описание клиента](#%s)", generateAnchor("Описание клиента")))
	md.LF()
	md.PlainText(markdown.Bold("Контракты:"))
	md.LF()

	for _, contract := range contracts {
		contractAnchor := contractAnchorID(contract.Name)
		md.PlainText(fmt.Sprintf("- [%s](#%s)", contract.Name, contractAnchor))
		md.LF()

		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			for _, method := range contract.Methods {
				if r.methodIsJsonRPC(contract, method) {
					methodAnchor := methodAnchorID(contract.Name, method.Name)
					md.PlainText(fmt.Sprintf("  - [%s](#%s)", method.Name, methodAnchor))
					md.LF()
				}
			}
		}

		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
			for _, method := range contract.Methods {
				if r.methodIsHTTP(contract, method) {
					httpMethod := model.GetHTTPMethod(r.project, contract, method)
					httpPath := model.GetAnnotationValue(r.project, contract, method, nil, model.TagHttpPath, "/"+ToLowerCamel(method.Name))
					methodTitle := fmt.Sprintf("%s %s", httpMethod, httpPath)
					methodAnchor := methodAnchorID(contract.Name, methodTitle)
					md.PlainText(fmt.Sprintf("  - [%s](#%s)", methodTitle, methodAnchor))
					md.LF()
				}
			}
		}
	}
	md.LF()

	if len(allTypes) > 0 {
		md.PlainText(markdown.Bold("Типы данных:"))
		md.LF()
		md.PlainText(fmt.Sprintf("- [Общие типы](#%s)", generateAnchor("Общие типы")))
		md.LF()
		for _, key := range typeKeys {
			usage := allTypes[key]
			typeAnchor := typeAnchorID(usage.fullTypeName)
			md.PlainText(fmt.Sprintf("  - [%s](#%s)", usage.fullTypeName, typeAnchor))
			md.LF()
		}
		md.LF()
	}

	md.PlainText(markdown.Bold("Вспомогательные разделы:"))
	md.LF()
	if hasJsonRPC {
		md.PlainText(fmt.Sprintf("- [Batch запросы (JSON-RPC)](#%s)", generateAnchor("Batch запросы (JSON-RPC)")))
		md.LF()
	}
	md.PlainText(fmt.Sprintf("- [Обработка ошибок](#%s)", generateAnchor("Обработка ошибок")))
	md.LF()
	md.PlainText(fmt.Sprintf("- [Логирование](#%s)", generateAnchor("Логирование")))
	md.LF()
	if r.HasMetrics() {
		md.PlainText(fmt.Sprintf("- [Метрики](#%s)", generateAnchor("Метрики")))
		md.LF()
	}
	md.HorizontalRule()

	r.renderClientDescription(md)

	for _, contract := range contracts {
		r.renderContract(md, contract, outDir, typeUsages)
	}

	if len(allTypes) > 0 {
		r.renderAllTypes(md, allTypes)
	}

	if hasJsonRPC {
		r.renderBatchSection(md, contracts, outDir)
	}

	r.renderErrorsSection(md)

	r.renderLoggingSection(md, outDir)

	if r.HasMetrics() {
		r.renderMetricsSection(md, outDir)
	}

	if err = md.Build(); err != nil {
		return err
	}

	outFilename := path.Join(outDir, "readme.md")
	if opts.FilePath != "" {
		outFilename = opts.FilePath
		readmeDir := path.Dir(outFilename)
		if err = os.MkdirAll(readmeDir, 0777); err != nil {
			return err
		}
	}

	return os.WriteFile(outFilename, buf.Bytes(), 0600)
}

func (r *ClientRenderer) renderClientDescription(md *markdown.Markdown) {
	md.H2("Описание клиента")
	md.PlainText("Go клиент для работы с API. Клиент поддерживает JSON-RPC и HTTP методы.")
	md.LF()
	md.PlainText("Основные возможности:")
	md.LF()
	capabilities := []string{
		"Поддержка JSON-RPC 2.0",
		"Поддержка HTTP методов (GET, POST, PUT, DELETE и др.)",
		"Upload/Download: потоковая передача тела запроса (io.Reader) и ответа (io.ReadCloser)",
		"Batch запросы для JSON-RPC",
		"Автоматическая обработка ошибок",
		"Типизированные методы для всех контрактов",
	}
	if r.HasMetrics() {
		capabilities = append(capabilities, "Поддержка Prometheus метрик")
	}
	md.BulletList(capabilities...)
	md.LF()

	var exampleContract *model.Contract
	var exampleMethod *model.Method

	for _, contractName := range r.ContractKeys() {
		contract := r.FindContract(contractName)
		if contract == nil {
			continue
		}
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			for _, method := range contract.Methods {
				if r.methodIsJsonRPC(contract, method) {
					exampleContract = contract
					exampleMethod = method
					break
				}
			}
			if exampleContract != nil {
				break
			}
		}
		if exampleContract == nil && model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
			for _, method := range contract.Methods {
				if r.methodIsHTTP(contract, method) {
					exampleContract = contract
					exampleMethod = method
					break
				}
			}
			if exampleContract != nil {
				break
			}
		}
	}

	pkgPath := r.pkgPath(r.outDir)
	pkgName := filepath.Base(r.outDir)
	if pkgName == "" || pkgName == "." {
		pkgName = "client"
	}

	md.PlainText(markdown.Bold("Инициализация клиента:"))
	md.LF()

	if exampleContract != nil && exampleMethod != nil {
		serviceVar := ToLowerCamel(exampleContract.Name)
		args := r.argsWithoutContext(exampleMethod)
		results := r.resultsWithoutError(exampleMethod)

		var methodCall string
		paramValues := make([]string, 0, len(args))
		for _, arg := range args {
			exampleValue := r.generateExampleValueFromVariable(arg, strings.Join(arg.Docs, "\n"), exampleContract.PkgPath)
			paramValues = append(paramValues, exampleValue)
		}

		ctxVar := "ctx"
		if len(paramValues) > 0 {
			methodCall = fmt.Sprintf("%s.%s(%s, %s)", serviceVar, exampleMethod.Name, ctxVar, strings.Join(paramValues, ", "))
		} else {
			methodCall = fmt.Sprintf("%s.%s(%s)", serviceVar, exampleMethod.Name, ctxVar)
		}

		var resultVar string
		if len(results) > 0 {
			resultVar = "result"
		}

		templateData := map[string]any{
			"PkgPath":      pkgPath,
			"PkgName":      pkgName,
			"ServiceVar":   serviceVar,
			"ContractName": exampleContract.Name,
			"MethodCall":   methodCall,
		}

		var codeExample string
		var err error
		if resultVar != "" {
			templateData["ResultVar"] = resultVar
			codeExample, err = r.renderTemplate("templates/simple_init_with_result.tmpl", templateData)
		} else {
			codeExample, err = r.renderTemplate("templates/simple_init_no_result.tmpl", templateData)
		}
		if err != nil {
			codeExample = fmt.Sprintf("// Error rendering template: %v", err)
		}
		md.CodeBlocks(markdown.SyntaxHighlightGo, codeExample)
	}
	md.LF()

	md.PlainText(markdown.Bold("Инициализация с опциями:"))
	md.LF()

	if exampleContract != nil && exampleMethod != nil {
		serviceVar := ToLowerCamel(exampleContract.Name)
		args := r.argsWithoutContext(exampleMethod)
		results := r.resultsWithoutError(exampleMethod)

		var methodCall string
		paramValues := make([]string, 0, len(args))
		for _, arg := range args {
			exampleValue := r.generateExampleValueFromVariable(arg, strings.Join(arg.Docs, "\n"), exampleContract.PkgPath)
			paramValues = append(paramValues, exampleValue)
		}

		if len(paramValues) > 0 {
			methodCall = fmt.Sprintf("%s.%s(context.Background(), %s)", serviceVar, exampleMethod.Name, strings.Join(paramValues, ", "))
		} else {
			methodCall = fmt.Sprintf("%s.%s(context.Background())", serviceVar, exampleMethod.Name)
		}

		var resultVar string
		if len(results) > 0 {
			resultVar = "result"
		}

		templateData := map[string]any{
			"PkgPath":      pkgPath,
			"PkgName":      pkgName,
			"ServiceVar":   serviceVar,
			"ContractName": exampleContract.Name,
			"MethodCall":   methodCall,
		}

		var codeExample string
		var err error
		if resultVar != "" {
			templateData["ResultVar"] = resultVar
			codeExample, err = r.renderTemplate("templates/init_with_options_with_result.tmpl", templateData)
		} else {
			codeExample, err = r.renderTemplate("templates/init_with_options_no_result.tmpl", templateData)
		}
		if err != nil {
			codeExample = fmt.Sprintf("// Error rendering template: %v", err)
		}
		md.CodeBlocks(markdown.SyntaxHighlightGo, codeExample)
	}
	md.LF()

	md.PlainText(markdown.Bold("Инициализация с кастомными заголовками:"))
	md.LF()

	if exampleContract != nil && exampleMethod != nil {
		serviceVar := ToLowerCamel(exampleContract.Name)
		args := r.argsWithoutContext(exampleMethod)
		results := r.resultsWithoutError(exampleMethod)

		var methodCall string
		paramValues := make([]string, 0, len(args))
		for _, arg := range args {
			exampleValue := r.generateExampleValueFromVariable(arg, strings.Join(arg.Docs, "\n"), exampleContract.PkgPath)
			paramValues = append(paramValues, exampleValue)
		}

		ctxVar := "ctx"
		if len(paramValues) > 0 {
			methodCall = fmt.Sprintf("%s.%s(%s, %s)", serviceVar, exampleMethod.Name, ctxVar, strings.Join(paramValues, ", "))
		} else {
			methodCall = fmt.Sprintf("%s.%s(%s)", serviceVar, exampleMethod.Name, ctxVar)
		}

		var resultVar string
		if len(results) > 0 {
			resultVar = "result"
		}

		templateData := map[string]any{
			"PkgPath":      pkgPath,
			"PkgName":      pkgName,
			"ServiceVar":   serviceVar,
			"ContractName": exampleContract.Name,
			"MethodName":   exampleMethod.Name,
			"MethodCall":   methodCall,
		}

		var codeExample string
		var err error
		if resultVar != "" {
			templateData["ResultVar"] = resultVar
			codeExample, err = r.renderTemplate("templates/init_with_headers_with_result.tmpl", templateData)
		} else {
			codeExample, err = r.renderTemplate("templates/init_with_headers_no_result.tmpl", templateData)
		}
		if err != nil {
			codeExample = fmt.Sprintf("// Error rendering template: %v", err)
		}
		md.CodeBlocks(markdown.SyntaxHighlightGo, codeExample)
	}
	md.LF()

	r.renderClientOptions(md, pkgPath, pkgName)

	md.HorizontalRule()
}

func (r *ClientRenderer) renderContract(md *markdown.Markdown, contract *model.Contract, outDir string, typeUsages map[string]*typeUsage) {
	contractAnchor := contractAnchorID(contract.Name)
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", contractAnchor))
	md.LF()
	md.H2(contract.Name)

	contractDesc := filterDocsComments(contract.Docs)
	if len(contractDesc) > 0 {
		md.PlainText(strings.Join(contractDesc, "\n"))
		md.LF()
	}

	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
		for _, method := range contract.Methods {
			if !r.methodIsJsonRPC(contract, method) {
				continue
			}
			r.renderMethodDoc(md, method, contract, outDir, typeUsages)
		}
	}

	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
		for _, method := range contract.Methods {
			if !r.methodIsHTTP(contract, method) {
				continue
			}
			r.renderHTTPMethodDoc(md, method, contract, outDir, typeUsages)
		}
	}

	md.HorizontalRule()
}

func generateAnchor(title string) string {
	anchor := strings.ToLower(title)

	anchor = strings.ReplaceAll(anchor, " ", "-")

	anchor = strings.ReplaceAll(anchor, "/", "-")
	anchor = strings.ReplaceAll(anchor, ":", "-")

	anchor = strings.ReplaceAll(anchor, "_", "-")

	for strings.Contains(anchor, "--") {
		anchor = strings.ReplaceAll(anchor, "--", "-")
	}

	anchor = strings.Trim(anchor, "-")

	if anchor == "" {
		anchor = "section"
	}

	return anchor
}

func contractAnchorID(contractName string) string {

	return "contract-" + generateAnchor(contractName)
}

func methodAnchorID(contractName string, methodNameOrTitle string) string {

	return contractAnchorID(contractName) + "-" + generateAnchor(methodNameOrTitle)
}

func typeAnchorID(typeName string) string {

	return "type-" + generateAnchor(typeName)
}

func (r *ClientRenderer) renderClientOptions(md *markdown.Markdown, pkgPath, pkgName string) {
	md.PlainText(markdown.Bold("Поддерживаемые опции клиента:"))
	md.LF()

	options := []struct {
		name        string
		description string
		signature   string
		example     string
	}{
		{
			name:        "DecodeError",
			description: "Устанавливает декодер ошибок для кастомной обработки ошибок",
			signature:   "func DecodeError(decoder ErrorDecoder) Option",
			example: fmt.Sprintf(`client := %s.New("http://localhost:9000",
    %s.DecodeError(customErrorDecoder),
)`, pkgName, pkgName),
		},
	}

	if r.HasJsonRPC() || r.HasHTTP() {
		options = append(options, []struct {
			name        string
			description string
			signature   string
			example     string
		}{
			{
				name:        "Headers",
				description: "Регистрирует ключи заголовков, значения которых будут браться из контекста при выполнении запросов. Значения заголовков должны быть установлены в контексте через context.WithValue",
				signature:   "func Headers(headers ...any) Option",
				example: fmt.Sprintf(`client := %s.New("http://localhost:9000",
    %s.Headers("Authorization", "X-API-Key"),
)

// При вызове методов передавайте контекст с значениями заголовков:
ctx := context.WithValue(context.Background(), "Authorization", "Bearer token123")
ctx = context.WithValue(ctx, "X-API-Key", "api-key-value")
result, err := service.Method(ctx, params...)`, pkgName, pkgName),
			},
			{
				name:        "ConfigTLS",
				description: "Настраивает TLS конфигурацию для HTTPS соединений",
				signature:   "func ConfigTLS(tlsConfig *tls.Config) Option",
				example: fmt.Sprintf(`tlsConfig := &tls.Config{
    InsecureSkipVerify: false,
}
client := %s.New("https://localhost:8443",
    %s.ConfigTLS(tlsConfig),
)`, pkgPath, pkgPath),
			},
			{
				name:        "LogRequest",
				description: "Включает логирование всех HTTP запросов на уровне Debug. Логи выводятся в структурированном формате через slog и содержат метод запроса и команду curl для воспроизведения запроса. Внимание: в логах могут содержаться чувствительные данные, включая заголовки авторизации, токены, пароли и тело запроса. Не используйте эту опцию в production окружении без дополнительной фильтрации чувствительных данных.",
				signature:   "func LogRequest() Option",
				example: fmt.Sprintf(`client := %s.New("http://localhost:9000",
    %s.LogRequest(),
)`, pkgName, pkgName),
			},
			{
				name:        "LogOnError",
				description: "Включает логирование только при ошибках на уровне Error. Логи выводятся в структурированном формате через slog и содержат метод запроса, команду curl для воспроизведения запроса и информацию об ошибке. Внимание: в логах могут содержаться чувствительные данные, включая заголовки авторизации, токены, пароли и тело запроса. Не используйте эту опцию в production окружении без дополнительной фильтрации чувствительных данных.",
				signature:   "func LogOnError() Option",
				example: fmt.Sprintf(`client := %s.New("http://localhost:9000",
    %s.LogOnError(),
)`, pkgName, pkgName),
			},
			{
				name:        "ClientHTTP",
				description: "Устанавливает кастомный HTTP клиент",
				signature:   "func ClientHTTP(client *http.Client) Option",
				example: fmt.Sprintf(`customClient := &http.Client{
    Timeout: 60 * time.Second,
}
client := %s.New("http://localhost:9000",
    %s.ClientHTTP(customClient),
)`, pkgName, pkgName),
			},
			{
				name:        "Transport",
				description: "Устанавливает кастомный HTTP транспорт",
				signature:   "func Transport(transport http.RoundTripper) Option",
				example: fmt.Sprintf(`transport := &http.Transport{
    MaxIdleConns: 100,
}
client := %s.New("http://localhost:9000",
    %s.Transport(transport),
)`, pkgName, pkgName),
			},
			{
				name:        "BeforeRequest",
				description: "Устанавливает функцию, вызываемую перед каждым запросом. Позволяет модифицировать запрос",
				signature:   "func BeforeRequest(before func(ctx context.Context, req *http.Request) context.Context) Option",
				example: fmt.Sprintf(`client := %s.New("http://localhost:9000",
    %s.BeforeRequest(func(ctx context.Context, req *http.Request) context.Context {
        req.Header.Set("X-Custom", "value")
        return ctx
    }),
)`, pkgName, pkgName),
			},
			{
				name:        "AfterRequest",
				description: "Устанавливает функцию, вызываемую после каждого запроса. Позволяет обработать ответ",
				signature:   "func AfterRequest(after func(ctx context.Context, res *http.Response) error) Option",
				example: fmt.Sprintf(`client := %s.New("http://localhost:9000",
    %s.AfterRequest(func(ctx context.Context, res *http.Response) error {
        // Обработка ответа
        return nil
    }),
)`, pkgName, pkgName),
			},
		}...)
	}

	if r.HasMetrics() {
		options = append(options, struct {
			name        string
			description string
			signature   string
			example     string
		}{
			name:        "WithMetrics",
			description: "Включает сбор Prometheus метрик",
			signature:   "func WithMetrics() Option",
			example: fmt.Sprintf(`client := %s.New("http://localhost:9000",
    %s.WithMetrics(),
)`, pkgName, pkgName),
		})
	}

	for _, opt := range options {
		md.PlainText(fmt.Sprintf("- **%s** - %s", markdown.Code(opt.name), opt.description))
		md.LF()
		md.PlainText(fmt.Sprintf("  Сигнатура: `%s`", opt.signature))
		md.LF()
		md.CodeBlocks(markdown.SyntaxHighlightGo, opt.example)
		md.LF()
	}
}

func (r *ClientRenderer) renderMetricsSection(md *markdown.Markdown, outDir string) {
	metricsAnchor := generateAnchor("Метрики")
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", metricsAnchor))
	md.LF()
	md.H2("Метрики")
	md.PlainText("Клиент поддерживает сбор Prometheus метрик для мониторинга работы API клиента.")
	md.LF()

	pkgPath := r.pkgPath(outDir)
	pkgName := filepath.Base(outDir)
	if pkgName == "" || pkgName == "." {
		pkgName = "client"
	}

	md.PlainText(markdown.Bold("Включение метрик:"))
	md.LF()
	md.CodeBlocks(markdown.SyntaxHighlightGo, fmt.Sprintf(`package main

import (
    "%s"
	"tgp/internal/model"
)

func main() {
    // Создаем клиент с метриками
    client := %s.New("http://localhost:9000",
        %s.WithMetrics(),
    )
    
    // Метрики автоматически собираются при выполнении запросов
}`, pkgPath, pkgName, pkgName))
	md.LF()

	md.PlainText(markdown.Bold("Доступные метрики:"))
	md.LF()

	metrics := []struct {
		name        string
		description string
		labels      string
	}{
		{
			name:        "client_versions_count",
			description: "Версии компонентов клиента",
			labels:      "part, version, hostname",
		},
		{
			name:        "client_requests_count",
			description: "Количество отправленных запросов",
			labels:      "service, method, success, errCode, client_id",
		},
		{
			name:        "client_requests_all_count",
			description: "Общее количество всех запросов",
			labels:      "service, method, success, errCode, client_id",
		},
		{
			name:        "client_requests_latency_seconds",
			description: "Задержка выполнения запросов в секундах",
			labels:      "service, method, success, errCode, client_id",
		},
	}

	for _, metric := range metrics {
		md.PlainText(fmt.Sprintf("- **%s** - %s", markdown.Code(metric.name), metric.description))
		md.LF()
		md.PlainText(fmt.Sprintf("  Метки: %s", markdown.Code(metric.labels)))
		md.LF()
	}

	md.PlainText(markdown.Bold("Экспорт метрик (один клиент):"))
	md.LF()
	md.CodeBlocks(markdown.SyntaxHighlightGo, `reg := client.GetMetricsRegistry()
if reg != nil {
    http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
}
http.ListenAndServe(":9090", nil)`)
	md.LF()
	md.PlainText(markdown.Bold("Несколько клиентов — объединение реестров в один /metrics:"))
	md.LF()
	md.CodeBlocks(markdown.SyntaxHighlightGo, `var gatherers []prometheus.Gatherer
if reg := clientA.GetMetricsRegistry(); reg != nil {
    gatherers = append(gatherers, reg)
}
if reg := clientB.GetMetricsRegistry(); reg != nil {
    gatherers = append(gatherers, reg)
}
if len(gatherers) > 0 {
    http.Handle("/metrics", promhttp.HandlerFor(prometheus.Gatherers(gatherers), promhttp.HandlerOpts{}))
}`)
	md.LF()
	md.HorizontalRule()
}

func filterDocsComments(docs []string) []string {
	if len(docs) == 0 {
		return docs
	}
	var filtered []string
	for _, doc := range docs {
		if !strings.Contains(doc, "@tg") {
			filtered = append(filtered, doc)
		}
	}
	return filtered
}

func (r *ClientRenderer) renderTemplate(templatePath string, data any) (string, error) {
	contentBytes, err := templatesFS.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	tmpl, err := template.New("").Funcs(template.FuncMap{
		"sub": func(a, b int) int {
			return a - b
		},
	}).Parse(string(contentBytes))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	return buf.String(), nil
}

func (r *ClientRenderer) renderLoggingSection(md *markdown.Markdown, outDir string) {
	loggingAnchor := generateAnchor("Логирование")
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", loggingAnchor))
	md.LF()
	md.H2("Логирование")
	md.PlainText("Клиент поддерживает структурированное логирование через стандартную библиотеку `log/slog`. Логирование позволяет отслеживать все HTTP/JSON-RPC запросы и ошибки, а также интегрировать логи клиента с существующей системой логирования приложения.")
	md.LF()

	pkgPath := r.pkgPath(outDir)
	pkgName := filepath.Base(outDir)
	if pkgName == "" || pkgName == "." {
		pkgName = "client"
	}

	md.PlainText(markdown.Bold("Структурированное логирование с slog:"))
	md.LF()
	md.PlainText("`log/slog` — это стандартная библиотека Go для структурированного логирования, которая предоставляет единый интерфейс для работы с различными бэкендами логирования. Основные преимущества структурированного логирования:")
	md.LF()
	md.BulletList(
		"Единый формат логов с ключ-значение парами, что упрощает парсинг и анализ",
		"Возможность фильтрации и поиска по структурированным полям",
		"Интеграция с системами мониторинга и анализа логов (ELK, Loki, Splunk и др.)",
		"Улучшенная читаемость и контекстность логов",
		"Поддержка уровней логирования (Debug, Info, Warn, Error)",
		"Гибкость в выборе формата вывода (JSON, текстовый) и бэкенда",
	)
	md.LF()

	md.PlainText(markdown.Bold("Бэкенды для slog:"))
	md.LF()
	md.PlainText("`slog` поддерживает различные бэкенды через реализацию интерфейса `slog.Handler`. В качестве бэкенда можно использовать:")
	md.LF()
	md.BulletList(
		markdown.Bold("zerolog")+" — один из самых эффективных и производительных бэкендов для slog. Обеспечивает высокую скорость записи логов и минимальные накладные расходы. Рекомендуется для production окружений с высокими нагрузками.",
		markdown.Bold("zap")+" — популярный структурированный логгер от Uber с высокой производительностью. Поддерживает различные форматы вывода и настройки производительности.",
		markdown.Bold("slog.NewJSONHandler")+" — стандартный JSON handler из библиотеки `log/slog`. Простой в использовании, подходит для большинства случаев.",
		markdown.Bold("slog.NewTextHandler")+" — текстовый handler для человекочитаемого формата. Удобен для разработки и отладки.",
		"Другие популярные бэкенды — можно использовать любую библиотеку логирования, которая реализует интерфейс `slog.Handler` (например, logrus и др.).",
	)
	md.LF()

	md.PlainText(markdown.Bold("Опции логирования:"))
	md.LF()
	md.BulletList(
		markdown.Bold("LogRequest()")+" - включает логирование всех HTTP/JSON-RPC запросов на уровне Debug. Каждый лог содержит метод запроса и команду curl для воспроизведения запроса.",
		markdown.Bold("LogOnError()")+" - включает логирование только при ошибках на уровне Error. Логи содержат метод запроса, команду curl и информацию об ошибке.",
	)
	md.LF()

	md.PlainText("Формат логов:")
	md.LF()
	md.PlainText("Логи выводятся в структурированном формате через `slog` и содержат следующие поля:")
	md.LF()
	md.BulletList(
		"`method` - HTTP метод или имя JSON-RPC метода",
		"`curl` - команда curl для воспроизведения запроса (включает URL, заголовки и тело запроса)",
		"`error` - информация об ошибке (только для LogOnError)",
		"`count` - количество запросов в batch (только для JSON-RPC batch запросов)",
	)
	md.LF()
	md.HorizontalRule()
	md.LF()

	md.H3("Инициализация логирования")
	md.PlainText("Для использования логирования необходимо настроить `slog` handler перед созданием клиента. Клиент использует глобальный logger через `slog.Default()`, поэтому все настройки handler применяются автоматически.")
	md.LF()

	templateData := map[string]any{
		"PkgPath": pkgPath,
		"PkgName": pkgName,
	}
	loggingInitExample, err := r.renderTemplate("templates/logging_init.tmpl", templateData)
	if err != nil {
		loggingInitExample = fmt.Sprintf("// Error rendering template: %v", err)
	}
	md.CodeBlocks(markdown.SyntaxHighlightGo, loggingInitExample)
	md.LF()

	md.H3("⚠️ Безопасность и рекомендации")
	md.PlainText("В логах могут содержаться чувствительные данные, включая:")
	md.LF()
	md.BulletList(
		"Заголовки авторизации (токены, API ключи)",
		"Пароли и секреты в теле запросов",
		"Персональные данные пользователей",
		"Внутренние URL и структуры данных",
	)
	md.LF()
	md.PlainText("Рекомендации:")
	md.LF()
	md.BulletList(
		"В production окружении используйте `LogOnError()` вместо `LogRequest()` для уменьшения объема логов",
		"Настройте уровень логирования через `slog.HandlerOptions.Level` для фильтрации по уровням",
		"Используйте кастомные handlers для фильтрации или маскировки чувствительных данных",
		"Не логируйте в production без необходимости или используйте отдельный logger с ограниченным доступом",
		"Рассмотрите возможность использования middleware для фильтрации чувствительных полей перед логированием",
	)
	md.LF()
	md.HorizontalRule()
}
