// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path"
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

func (r *ClientRenderer) RenderReadmeTS(docOpts DocOptions) error {
	var err error
	outDir := r.outDir

	if !docOpts.Enabled {
		return nil
	}

	var buf bytes.Buffer
	md := markdown.NewMarkdown(&buf)

	md.H1("API Документация")
	md.PlainText("Автоматически сгенерированная документация API для TypeScript клиента.")

	contracts := make([]*model.Contract, len(r.project.Contracts))
	copy(contracts, r.project.Contracts)
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].Name < contracts[j].Name
	})

	hasJsonRPC := false

	md.H2("Оглавление")
	md.PlainText(fmt.Sprintf("- [Описание клиента](#%s)", generateAnchor("Описание клиента")))
	md.LF()
	md.PlainText(markdown.Bold("Контракты:"))
	md.LF()

	for _, contract := range contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			hasJsonRPC = true
		}

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
				if r.methodIsHTTP(method, contract) {
					httpMethod := model.GetHTTPMethod(r.project, contract, method)
					httpPath := model.GetAnnotationValue(r.project, contract, method, nil, model.TagHttpPath, "")
					methodTitle := fmt.Sprintf("%s %s", httpMethod, httpPath)
					methodAnchor := methodAnchorID(contract.Name, methodTitle)
					md.PlainText(fmt.Sprintf("  - [%s](#%s)", methodTitle, methodAnchor))
					md.LF()
				}
			}
		}
	}
	md.LF()

	typeUsages := r.collectStructTypesTS()
	allTypes := make(map[string]*typeUsageTS)
	for key, usage := range common.SortedPairs(typeUsages) {
		allTypes[key] = usage
	}

	r.typeAnchorsSet = make(map[string]bool)
	for _, usage := range allTypes {
		r.typeAnchorsSet[typeAnchorID(usage.fullTypeName)] = true
	}

	typeKeys := common.SortedKeys(allTypes)
	sort.Strings(typeKeys)

	if len(allTypes) > 0 {
		md.PlainText(markdown.Bold("Типы данных:"))
		md.LF()
		md.PlainText(fmt.Sprintf("- [Общие типы](#%s)", generateAnchor("Общие типы")))
		md.LF()

		typesByNamespaceTOC := make(map[string][]*typeUsageTS)
		for _, usage := range common.SortedPairs(allTypes) {
			namespace := ""
			if strings.Contains(usage.fullTypeName, ".") {
				parts := strings.SplitN(usage.fullTypeName, ".", 2)
				namespace = parts[0]
			}
			if typesByNamespaceTOC[namespace] == nil {
				typesByNamespaceTOC[namespace] = make([]*typeUsageTS, 0)
			}
			typesByNamespaceTOC[namespace] = append(typesByNamespaceTOC[namespace], usage)
		}

		namespaceKeysTOC := make([]string, 0, len(typesByNamespaceTOC))
		for ns := range typesByNamespaceTOC {
			namespaceKeysTOC = append(namespaceKeysTOC, ns)
		}
		sort.Strings(namespaceKeysTOC)

		for _, namespace := range namespaceKeysTOC {
			types := typesByNamespaceTOC[namespace]
			sort.Slice(types, func(i, j int) bool {
				return types[i].fullTypeName < types[j].fullTypeName
			})

			if namespace != "" {
				namespaceAnchor := generateAnchor(namespace)
				md.PlainText(fmt.Sprintf("  - [%s](#%s)", namespace, namespaceAnchor))
				md.LF()
			}

			for _, usage := range types {
				var typ *model.Type
				for _, t := range common.SortedPairs(r.project.Types) {
					if t.TypeName == usage.typeName {
						tPkg := t.ImportPkgPath
						if tPkg == "" {
							tPkg = usage.pkgPath
						}
						if tPkg == usage.pkgPath {
							typ = t
							break
						}
					}
				}
				if typ != nil {
					if r.isExplicitlyExcludedType(typ) || r.hasMarshaler(typ, true) || r.hasMarshaler(typ, false) {
						continue
					}
				}
				typeAnchor := typeAnchorID(usage.fullTypeName)
				md.PlainText(fmt.Sprintf("    - [%s](#%s)", usage.fullTypeName, typeAnchor))
				md.LF()
			}
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
	md.HorizontalRule()

	r.renderClientDescriptionTS(md)

	for _, contract := range contracts {
		r.renderContractTS(md, contract, outDir)
	}

	if len(allTypes) > 0 {
		r.renderAllTypesTS(md, allTypes)
	}

	if hasJsonRPC {
		r.renderBatchSectionTS(md, contracts, outDir)
	}

	r.renderErrorsSectionTS(md, outDir)

	if err = md.Build(); err != nil {
		return err
	}

	outFilename := path.Join(outDir, "readme.md")
	if docOpts.FilePath != "" {
		outFilename = docOpts.FilePath
		readmeDir := path.Dir(outFilename)
		if err = os.MkdirAll(readmeDir, 0777); err != nil {
			return err
		}
	}

	return os.WriteFile(outFilename, buf.Bytes(), 0600)
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

func (r *ClientRenderer) methodIsJsonRPC(contract *model.Contract, method *model.Method) bool {

	if method == nil {
		return false
	}
	return contract != nil && model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) && !model.IsAnnotationSet(r.project, contract, method, nil, model.TagHTTPMethod)
}

func (r *ClientRenderer) methodIsHTTP(method *model.Method, contract *model.Contract) bool {

	return r.isHTTP(method, contract)
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

func (r *ClientRenderer) renderClientDescriptionTS(md *markdown.Markdown) {
	md.H2("Описание клиента")
	md.PlainText("TypeScript клиент для работы с API. Клиент поддерживает JSON-RPC и HTTP методы.")
	md.LF()
	md.PlainText("Основные возможности:")
	md.LF()
	capabilities := []string{
		"Поддержка JSON-RPC 2.0",
		"Поддержка HTTP методов (GET, POST, PUT, DELETE и др.)",
		"Отправка и приём бинарных данных (Blob, FormData для multipart)",
		"Batch запросы для JSON-RPC",
		"Автоматическая обработка ошибок",
		"Типизированные методы для всех контрактов",
	}
	md.BulletList(capabilities...)
	md.LF()

	var exampleContract *model.Contract
	var exampleMethod *model.Method

	contracts := make([]*model.Contract, len(r.project.Contracts))
	copy(contracts, r.project.Contracts)
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].Name < contracts[j].Name
	})
	for _, contract := range contracts {
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
				if r.methodIsHTTP(method, contract) {
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

	md.PlainText(markdown.Bold("Инициализация клиента:"))
	md.LF()

	if exampleContract != nil && exampleMethod != nil {
		fileName := r.tsFileName(exampleContract)
		parts := strings.Split(fileName, "_")
		serviceVar := ""
		for i, part := range parts {
			if i == 0 {
				serviceVar += part
			} else if len(part) > 0 {
				serviceVar += strings.ToUpper(string(part[0])) + part[1:]
			}
		}
		if exampleContract.Annotations.IsSet(model.TagServerHTTP) {
			serviceVar += "HTTP"
		}
		args := r.argsWithoutContext(exampleMethod)
		results := r.resultsWithoutError(exampleMethod)

		var methodCall string
		paramValues := make([]string, 0, len(args))
		for _, arg := range args {
			exampleValue := r.generateExampleValueFromVariable(arg, strings.Join(arg.Docs, "\n"), exampleContract.PkgPath)
			paramValues = append(paramValues, exampleValue)
		}

		if len(paramValues) > 0 {
			methodCall = fmt.Sprintf("%s.%s(%s)", serviceVar, r.lcName(exampleMethod.Name), strings.Join(paramValues, ", "))
		} else {
			methodCall = fmt.Sprintf("%s.%s()", serviceVar, r.lcName(exampleMethod.Name))
		}

		var resultVar string
		if len(results) > 0 {
			resultVar = "result"
		}

		clientMethodName := r.getClientMethodName(exampleContract)

		templateData := map[string]any{
			"ServiceVar":       serviceVar,
			"ClientMethodName": clientMethodName,
			"MethodCall":       methodCall,
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
		md.CodeBlocks(markdown.SyntaxHighlightTypeScript, codeExample)
	}
	md.LF()

	md.PlainText(markdown.Bold("Инициализация с опциями:"))
	md.LF()

	if exampleContract != nil && exampleMethod != nil {
		fileName := r.tsFileName(exampleContract)
		parts := strings.Split(fileName, "_")
		serviceVar := ""
		for i, part := range parts {
			if i == 0 {
				serviceVar += part
			} else if len(part) > 0 {
				serviceVar += strings.ToUpper(string(part[0])) + part[1:]
			}
		}
		if exampleContract.Annotations.IsSet(model.TagServerHTTP) {
			serviceVar += "HTTP"
		}
		args := r.argsWithoutContext(exampleMethod)
		results := r.resultsWithoutError(exampleMethod)

		var methodCall string
		paramValues := make([]string, 0, len(args))
		for _, arg := range args {
			exampleValue := r.generateExampleValueFromVariable(arg, strings.Join(arg.Docs, "\n"), exampleContract.PkgPath)
			paramValues = append(paramValues, exampleValue)
		}

		if len(paramValues) > 0 {
			methodCall = fmt.Sprintf("%s.%s(%s)", serviceVar, r.lcName(exampleMethod.Name), strings.Join(paramValues, ", "))
		} else {
			methodCall = fmt.Sprintf("%s.%s()", serviceVar, r.lcName(exampleMethod.Name))
		}

		var resultVar string
		if len(results) > 0 {
			resultVar = "result"
		}

		clientMethodName := r.getClientMethodName(exampleContract)

		templateData := map[string]any{
			"ServiceVar":       serviceVar,
			"ClientMethodName": clientMethodName,
			"MethodCall":       methodCall,
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
		md.CodeBlocks(markdown.SyntaxHighlightTypeScript, codeExample)
	}
	md.LF()

	md.PlainText(markdown.Bold("Инициализация с кастомными заголовками:"))
	md.LF()

	if exampleContract != nil && exampleMethod != nil {
		fileName := r.tsFileName(exampleContract)
		parts := strings.Split(fileName, "_")
		serviceVar := ""
		for i, part := range parts {
			if i == 0 {
				serviceVar += part
			} else if len(part) > 0 {
				serviceVar += strings.ToUpper(string(part[0])) + part[1:]
			}
		}
		if exampleContract.Annotations.IsSet(model.TagServerHTTP) {
			serviceVar += "HTTP"
		}
		args := r.argsWithoutContext(exampleMethod)
		results := r.resultsWithoutError(exampleMethod)

		var methodCall string
		paramValues := make([]string, 0, len(args))
		for _, arg := range args {
			exampleValue := r.generateExampleValueFromVariable(arg, strings.Join(arg.Docs, "\n"), exampleContract.PkgPath)
			paramValues = append(paramValues, exampleValue)
		}

		if len(paramValues) > 0 {
			methodCall = fmt.Sprintf("%s.%s(%s)", serviceVar, r.lcName(exampleMethod.Name), strings.Join(paramValues, ", "))
		} else {
			methodCall = fmt.Sprintf("%s.%s()", serviceVar, r.lcName(exampleMethod.Name))
		}

		var resultVar string
		if len(results) > 0 {
			resultVar = "result"
		}

		clientMethodName := r.getClientMethodName(exampleContract)

		templateData := map[string]any{
			"ServiceVar":       serviceVar,
			"ClientMethodName": clientMethodName,
			"MethodName":       exampleMethod.Name,
			"MethodCall":       methodCall,
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
		md.CodeBlocks(markdown.SyntaxHighlightTypeScript, codeExample)
	}
	md.LF()

	md.PlainText(markdown.Bold("Инициализация с функциональными заголовками (динамические токены):"))
	md.LF()
	md.PlainText("Для случаев, когда заголовки должны вычисляться при каждом запросе (например, токены авторизации), можно использовать функцию:")
	md.LF()

	if exampleContract != nil && exampleMethod != nil {
		fileName := r.tsFileName(exampleContract)
		parts := strings.Split(fileName, "_")
		serviceVar := ""
		for i, part := range parts {
			if i == 0 {
				serviceVar += part
			} else if len(part) > 0 {
				serviceVar += strings.ToUpper(string(part[0])) + part[1:]
			}
		}
		if exampleContract.Annotations.IsSet(model.TagServerHTTP) {
			serviceVar += "HTTP"
		}
		args := r.argsWithoutContext(exampleMethod)
		results := r.resultsWithoutError(exampleMethod)

		var methodCall string
		paramValues := make([]string, 0, len(args))
		for _, arg := range args {
			exampleValue := r.generateExampleValueFromVariable(arg, strings.Join(arg.Docs, "\n"), exampleContract.PkgPath)
			paramValues = append(paramValues, exampleValue)
		}

		if len(paramValues) > 0 {
			methodCall = fmt.Sprintf("%s.%s(%s)", serviceVar, r.lcName(exampleMethod.Name), strings.Join(paramValues, ", "))
		} else {
			methodCall = fmt.Sprintf("%s.%s()", serviceVar, r.lcName(exampleMethod.Name))
		}

		var resultVar string
		if len(results) > 0 {
			resultVar = "result"
		}

		clientMethodName := r.getClientMethodName(exampleContract)

		templateData := map[string]any{
			"ServiceVar":       serviceVar,
			"ClientMethodName": clientMethodName,
			"MethodName":       exampleMethod.Name,
			"MethodCall":       methodCall,
		}

		var codeExample string
		var err error
		if resultVar != "" {
			templateData["ResultVar"] = resultVar
			codeExample, err = r.renderTemplate("templates/init_with_dynamic_headers_with_result.tmpl", templateData)
		} else {
			codeExample, err = r.renderTemplate("templates/init_with_dynamic_headers_no_result.tmpl", templateData)
		}
		if err != nil {
			codeExample = fmt.Sprintf("// Error rendering template: %v", err)
		}
		md.CodeBlocks(markdown.SyntaxHighlightTypeScript, codeExample)
	}
	md.LF()
	md.PlainText("Функция заголовков может быть синхронной или асинхронной (возвращать Promise). При каждом запросе функция будет вызвана, что позволяет использовать актуальные токены авторизации.")
	md.LF()
	md.HorizontalRule()
}

func (r *ClientRenderer) renderContractTS(md *markdown.Markdown, contract *model.Contract, outDir string) {
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
			r.renderMethodDocTS(md, method, contract, outDir)
		}
	}

	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
		for _, method := range contract.Methods {
			if !r.methodIsHTTP(method, contract) {
				continue
			}
			r.renderHTTPMethodDocTS(md, method, contract, outDir)
		}
	}

	md.HorizontalRule()
}

func (r *ClientRenderer) renderMethodDocTS(md *markdown.Markdown, method *model.Method, contract *model.Contract, outDir string) {
	methodAnchor := methodAnchorID(contract.Name, method.Name)
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", methodAnchor))
	md.LF()
	md.H3(method.Name)

	summary := model.GetAnnotationValue(r.project, contract, method, nil, tagSummary, "")
	methodDesc := filterDocsComments(method.Docs)
	desc := ""
	if len(methodDesc) > 0 {
		desc = methodDesc[0]
	}
	if summary == "" && desc != "" {
		summary = desc
	}
	if summary == "" {
		summary = method.Name
	}

	if summary != "" {
		md.PlainText(markdown.Bold("Описание:") + " " + summary)
		md.LF()
	}
	if desc != "" && desc != summary {
		md.PlainText(desc)
		md.LF()
	}

	r.renderMethodSignatureTS(md, method, contract, false)

	r.renderMethodParamsAndResultsTS(md, method, contract)

	r.renderMethodErrorsTS(md, method, contract)

	md.LF()
}

func (r *ClientRenderer) renderHTTPMethodDocTS(md *markdown.Markdown, method *model.Method, contract *model.Contract, outDir string) {

	httpMethod := model.GetHTTPMethod(r.project, contract, method)
	httpPath := model.GetAnnotationValue(r.project, contract, method, nil, model.TagHttpPath, "")

	methodTitle := fmt.Sprintf("%s %s", httpMethod, httpPath)
	methodAnchor := methodAnchorID(contract.Name, methodTitle)
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", methodAnchor))
	md.LF()
	md.H3(methodTitle)

	summary := model.GetAnnotationValue(r.project, contract, method, nil, tagSummary, "")
	methodDesc := filterDocsComments(method.Docs)
	desc := ""
	if len(methodDesc) > 0 {
		desc = methodDesc[0]
	}
	if summary == "" && desc != "" {
		summary = desc
	}
	if summary == "" {
		summary = method.Name
	}

	if summary != "" {
		md.PlainText(markdown.Bold("Описание:") + " " + summary)
		md.LF()
	}
	if desc != "" && desc != summary {
		md.PlainText(desc)
		md.LF()
	}

	if r.methodRequestMultipart(contract, method) {
		streamArgs := r.methodRequestBodyStreamArgs(method)
		parts := make([]string, 0, len(streamArgs))
		for _, arg := range streamArgs {
			parts = append(parts, markdown.Code(arg.Name))
		}
		md.PlainText(markdown.Bold("Upload (multipart):") + " тело запроса передаётся как " + markdown.Code("FormData") + " (multipart/form-data), части: " + strings.Join(parts, ", ") + ". Имя и Content-Type части — аннотации " + markdown.Code("http-part-name") + ", " + markdown.Code("http-part-content") + ".")
		md.LF()
	} else if bodyStreamArg := r.methodRequestBodyStreamArg(method); bodyStreamArg != nil {
		md.PlainText(markdown.Bold("Upload:") + " тело запроса передаётся как " + markdown.Code("Blob") + " (аргумент " + markdown.Code(bodyStreamArg.Name) + "). Content-Type из аннотации " + markdown.Code("requestContentType") + " или " + markdown.Code("application/octet-stream") + ".")
		md.LF()
	}
	if r.methodResponseMultipart(contract, method) {
		streamResults := r.methodResponseBodyStreamResults(method)
		parts := make([]string, 0, len(streamResults))
		for _, res := range streamResults {
			parts = append(parts, markdown.Code(res.Name))
		}
		md.PlainText(markdown.Bold("Download (multipart):") + " тело ответа возвращается как " + markdown.Code("FormData") + " (части: " + strings.Join(parts, ", ") + "), значения частей — " + markdown.Code("Blob") + ".")
		md.LF()
	} else if responseStreamResult := r.methodResponseBodyStreamResult(method); responseStreamResult != nil {
		md.PlainText(markdown.Bold("Download:") + " тело ответа возвращается как " + markdown.Code("Blob") + " (" + markdown.Code(responseStreamResult.Name) + ").")
		md.LF()
	}

	r.renderMethodSignatureTS(md, method, contract, true)

	r.renderMethodParamsAndResultsTS(md, method, contract)

	r.renderMethodErrorsTS(md, method, contract)

	md.LF()
}

func (r *ClientRenderer) renderMethodSignatureTS(md *markdown.Markdown, method *model.Method, contract *model.Contract, isHTTP bool) {
	md.PlainText(markdown.Bold("Сигнатура:"))
	md.LF()

	var sigBuilder strings.Builder
	fileName := r.tsFileName(contract)
	parts := strings.Split(fileName, "_")
	serviceVar := ""
	for i, part := range parts {
		if i == 0 {
			serviceVar += part
		} else if len(part) > 0 {
			serviceVar += strings.ToUpper(string(part[0])) + part[1:]
		}
	}
	if isHTTP {
		serviceVar += "HTTP"
	}
	sigBuilder.WriteString(serviceVar)
	sigBuilder.WriteString(".")
	sigBuilder.WriteString(r.lcName(method.Name))
	sigBuilder.WriteString("(")

	args := r.argsWithoutContext(method)
	if len(args) > 0 {
		for i, arg := range args {
			if i > 0 {
				sigBuilder.WriteString(", ")
			}
			sigBuilder.WriteString(tsSafeName(arg.Name))
			sigBuilder.WriteString(": ")
			typeStr := r.tsTypeStringFromVariable(arg, contract.PkgPath)
			sigBuilder.WriteString(typeStr)
		}
	}

	sigBuilder.WriteString("): Promise<")

	results := r.resultsWithoutError(method)
	if len(results) > 0 {
		if len(results) == 1 {
			typeStr := r.tsTypeStringFromVariable(results[0], contract.PkgPath)
			sigBuilder.WriteString(typeStr)
		} else {
			sigBuilder.WriteString("{ ")
			for i, result := range results {
				if i > 0 {
					sigBuilder.WriteString(", ")
				}
				resultName := tsSafeName(result.Name)
				sigBuilder.WriteString(resultName)
				sigBuilder.WriteString(": ")
				typeStr := r.tsTypeStringFromVariable(result, contract.PkgPath)
				sigBuilder.WriteString(typeStr)
			}
			sigBuilder.WriteString(" }")
		}
	} else {
		sigBuilder.WriteString("void")
	}

	sigBuilder.WriteString(">")

	md.CodeBlocks(markdown.SyntaxHighlightTypeScript, sigBuilder.String())
	md.LF()
}

func (r *ClientRenderer) renderMethodParamsAndResultsTS(md *markdown.Markdown, method *model.Method, contract *model.Contract) {
	args := r.argsWithoutContext(method)
	results := r.resultsWithoutError(method)

	if len(args) > 0 {
		md.PlainText(markdown.Bold("Параметры:"))
		md.LF()

		rows := make([][]string, 0, len(args))
		for _, arg := range args {
			typeStr := r.tsTypeStringFromVariable(arg, contract.PkgPath)
			typeLink := r.getTypeLinkFromVariableTS(arg, contract.PkgPath)

			rows = append(rows, []string{
				markdown.Code(tsSafeName(arg.Name)),
				markdown.Code(typeStr),
				typeLink,
			})
		}

		headers := []string{"Имя", "Тип", "Ссылка на тип"}
		tableSet := markdown.TableSet{
			Header: headers,
			Rows:   rows,
		}
		md.Table(tableSet)
		md.LF()
	}

	if len(results) > 0 {
		md.PlainText(markdown.Bold("Возвращаемые значения:"))
		md.LF()

		rows := make([][]string, 0, len(results))
		for _, result := range results {
			typeStr := r.tsTypeStringFromVariable(result, contract.PkgPath)
			typeLink := r.getTypeLinkFromVariableTS(result, contract.PkgPath)

			resultName := tsSafeName(result.Name)

			rows = append(rows, []string{
				markdown.Code(resultName),
				markdown.Code(typeStr),
				typeLink,
			})
		}

		headers := []string{"Имя", "Тип", "Ссылка на тип"}
		tableSet := markdown.TableSet{
			Header: headers,
			Rows:   rows,
		}
		md.Table(tableSet)
		md.LF()
	}
}

func (r *ClientRenderer) renderMethodErrorsTS(md *markdown.Markdown, method *model.Method, contract *model.Contract) {
	if len(method.Errors) == 0 {
		return
	}

	md.LF()
	md.PlainText(markdown.Bold("Возможные ошибки:"))
	md.LF()

	errors := make([]*model.ErrorInfo, len(method.Errors))
	copy(errors, method.Errors)
	sort.Slice(errors, func(i, j int) bool {
		if errors[i].HTTPCode == 0 && errors[j].HTTPCode != 0 {
			return false
		}
		if errors[i].HTTPCode != 0 && errors[j].HTTPCode == 0 {
			return true
		}
		return errors[i].HTTPCode < errors[j].HTTPCode
	})

	for _, errInfo := range errors {
		if errInfo.HTTPCode != 0 {
			errorDesc := fmt.Sprintf("%s (%d)", errInfo.HTTPCodeText, errInfo.HTTPCode)
			md.PlainText(fmt.Sprintf("- %s - %s", markdown.Code(fmt.Sprintf("%d", errInfo.HTTPCode)), errorDesc))
		} else {
			md.PlainText(fmt.Sprintf("- %s", markdown.Code(errInfo.TypeName)))
		}
		md.LF()
	}
}

func (r *ClientRenderer) renderBatchSectionTS(md *markdown.Markdown, contracts []*model.Contract, outDir string) {
	batchAnchor := generateAnchor("Batch запросы (JSON-RPC)")
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", batchAnchor))
	md.LF()
	md.H2("Batch запросы (JSON-RPC)")
	md.PlainText("Для выполнения нескольких JSON-RPC запросов одновременно используйте метод " + markdown.Code("batch") + ". Это позволяет отправить несколько запросов в одном HTTP запросе.")
	md.LF()

	md.PlainText(markdown.Bold("Пример использования:"))
	md.LF()

	r.renderBatchExampleTS(md, contracts, outDir)
	md.LF()

	md.HorizontalRule()
}

func (r *ClientRenderer) renderBatchExampleTS(md *markdown.Markdown, contracts []*model.Contract, outDir string) {
	var jsonRPCContracts []*model.Contract
	for _, contract := range contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			hasJsonRPCMethods := false
			for _, method := range contract.Methods {
				if r.methodIsJsonRPC(contract, method) {
					hasJsonRPCMethods = true
					break
				}
			}
			if hasJsonRPCMethods {
				jsonRPCContracts = append(jsonRPCContracts, contract)
			}
		}
	}

	if len(jsonRPCContracts) == 0 {
		codeExample := `import { Client } from './client';

const client = new Client('http://localhost:9000');

await client.batch(
  service.reqMethod1(callback1, param1, param2),
  service.reqMethod2(callback2, param3)
);`
		md.CodeBlocks(markdown.SyntaxHighlightTypeScript, codeExample)
		return
	}

	exampleContract := jsonRPCContracts[0]
	var exampleMethod1 *model.Method
	var exampleMethod2 *model.Method
	for _, method := range exampleContract.Methods {
		if r.methodIsJsonRPC(exampleContract, method) {
			if exampleMethod1 == nil {
				exampleMethod1 = method
			} else {
				exampleMethod2 = method
				break
			}
		}
	}

	// Fallback: второй метод из другого контракта, если в первом только один JSON-RPC метод.
	if exampleMethod2 == nil && len(jsonRPCContracts) > 1 {
		for _, contract := range jsonRPCContracts[1:] {
			for _, method := range contract.Methods {
				if r.methodIsJsonRPC(contract, method) {
					exampleMethod2 = method
					break
				}
			}
			if exampleMethod2 != nil {
				break
			}
		}
	}

	var codeBuilder strings.Builder
	codeBuilder.WriteString("import { Client } from './client';\n\n")
	codeBuilder.WriteString("const client = new Client('http://localhost:9000');\n\n")

	serviceVar := r.getClientMethodName(exampleContract)
	codeBuilder.WriteString(fmt.Sprintf("const %s = client.%s();\n\n", serviceVar, serviceVar))

	// Порядок: callback-функции, затем массив вызовов reqXxx() — как в сгенерированном клиенте.
	if exampleMethod1 != nil {
		results1 := r.resultsWithoutError(exampleMethod1)
		callbackName1 := "callback1"
		callbackParams1 := make([]string, 0)
		for _, ret := range results1 {
			typeStr := r.walkVariable(ret.Name, exampleContract.PkgPath, ret, exampleMethod1.Annotations, false).typeLink()
			callbackParams1 = append(callbackParams1, fmt.Sprintf("%s: %s", tsSafeName(ret.Name), typeStr))
		}
		callbackParams1 = append(callbackParams1, "error: Error | null")
		codeBuilder.WriteString(fmt.Sprintf("function %s(%s) {\n", callbackName1, strings.Join(callbackParams1, ", ")))
		codeBuilder.WriteString("  if (error) {\n")
		codeBuilder.WriteString(fmt.Sprintf("    console.error('Error in %s.%s:', error);\n", exampleContract.Name, exampleMethod1.Name))
		codeBuilder.WriteString("    return;\n")
		codeBuilder.WriteString("  }\n")
		if len(results1) > 0 {
			resultVar := tsSafeName(results1[0].Name)
			codeBuilder.WriteString(fmt.Sprintf("  console.log('Result:', %s);\n", resultVar))
		}
		codeBuilder.WriteString("}\n\n")
	}
	if exampleMethod2 != nil {
		results2 := r.resultsWithoutError(exampleMethod2)
		callbackName2 := "callback2"
		callbackParams2 := make([]string, 0)
		for _, ret := range results2 {
			typeStr := r.walkVariable(ret.Name, exampleContract.PkgPath, ret, exampleMethod2.Annotations, false).typeLink()
			callbackParams2 = append(callbackParams2, fmt.Sprintf("%s: %s", tsSafeName(ret.Name), typeStr))
		}
		callbackParams2 = append(callbackParams2, "error: Error | null")
		codeBuilder.WriteString(fmt.Sprintf("function %s(%s) {\n", callbackName2, strings.Join(callbackParams2, ", ")))
		codeBuilder.WriteString("  if (error) {\n")
		codeBuilder.WriteString(fmt.Sprintf("    console.error('Error in %s.%s:', error);\n", exampleContract.Name, exampleMethod2.Name))
		codeBuilder.WriteString("    return;\n")
		codeBuilder.WriteString("  }\n")
		if len(results2) > 0 {
			resultVar := tsSafeName(results2[0].Name)
			codeBuilder.WriteString(fmt.Sprintf("  console.log('Result:', %s);\n", resultVar))
		}
		codeBuilder.WriteString("}\n\n")
	}

	codeBuilder.WriteString("// Создаем batch запрос с несколькими методами\n")
	codeBuilder.WriteString("const requests = [\n")

	if exampleMethod1 != nil {
		args1 := r.argsWithoutContext(exampleMethod1)
		callbackName1 := "callback1"
		var paramValues1 []string
		for _, arg := range args1 {
			exampleValue := r.generateExampleValueFromVariable(arg, strings.Join(arg.Docs, "\n"), exampleContract.PkgPath)
			paramValues1 = append(paramValues1, exampleValue)
		}
		reqCall1 := fmt.Sprintf("  %s.req%s(%s", serviceVar, exampleMethod1.Name, callbackName1)
		if len(paramValues1) > 0 {
			reqCall1 += ", " + strings.Join(paramValues1, ", ")
		}
		reqCall1 += "),"
		codeBuilder.WriteString(reqCall1 + "\n")
	}
	if exampleMethod2 != nil {
		args2 := r.argsWithoutContext(exampleMethod2)
		callbackName2 := "callback2"
		var paramValues2 []string
		for _, arg := range args2 {
			exampleValue := r.generateExampleValueFromVariable(arg, strings.Join(arg.Docs, "\n"), exampleContract.PkgPath)
			paramValues2 = append(paramValues2, exampleValue)
		}
		reqCall2 := fmt.Sprintf("  %s.req%s(%s", serviceVar, exampleMethod2.Name, callbackName2)
		if len(paramValues2) > 0 {
			reqCall2 += ", " + strings.Join(paramValues2, ", ")
		}
		reqCall2 += "),"
		codeBuilder.WriteString(reqCall2 + "\n")
	}

	codeBuilder.WriteString("];\n\n")
	codeBuilder.WriteString("// Выполняем batch запрос\n")
	codeBuilder.WriteString("await client.batch(requests);\n\n")
	codeBuilder.WriteString("// Callback функции будут вызваны автоматически при получении ответов\n")

	md.CodeBlocks(markdown.SyntaxHighlightTypeScript, codeBuilder.String())
}

func (r *ClientRenderer) renderErrorsSectionTS(md *markdown.Markdown, outDir string) {
	errorsAnchor := generateAnchor("Обработка ошибок")
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", errorsAnchor))
	md.LF()
	md.H2("Обработка ошибок")

	var jsonrpcContract *model.Contract
	var jsonrpcMethod *model.Method
	contracts := make([]*model.Contract, len(r.project.Contracts))
	copy(contracts, r.project.Contracts)
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].Name < contracts[j].Name
	})
	for _, contract := range contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			for _, method := range contract.Methods {
				if r.methodIsJsonRPC(contract, method) {
					jsonrpcContract = contract
					jsonrpcMethod = method
					break
				}
			}
			if jsonrpcContract != nil {
				break
			}
		}
	}

	jsonrpcErrorsAnchor := generateAnchor("JSON-RPC ошибки")
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", jsonrpcErrorsAnchor))
	md.LF()
	md.H3("JSON-RPC ошибки")
	md.PlainText("При работе с JSON-RPC клиентом ошибки обрабатываются автоматически. Если сервер возвращает ошибку в формате JSON-RPC 2.0, клиент выбрасывает исключение.")

	var codeExample string
	if jsonrpcContract != nil && jsonrpcMethod != nil {
		fileName := r.tsFileName(jsonrpcContract)
		parts := strings.Split(fileName, "_")
		serviceVar := ""
		for i, part := range parts {
			if i == 0 {
				serviceVar += part
			} else if len(part) > 0 {
				serviceVar += strings.ToUpper(string(part[0])) + part[1:]
			}
		}
		clientMethodName := r.getClientMethodName(jsonrpcContract)
		methodName := r.lcName(jsonrpcMethod.Name)
		args := r.argsWithoutContext(jsonrpcMethod)

		paramValues := make([]string, 0, len(args))
		for _, arg := range args {
			exampleValue := r.generateExampleValueFromVariable(arg, strings.Join(arg.Docs, "\n"), jsonrpcContract.PkgPath)
			paramValues = append(paramValues, exampleValue)
		}

		var methodCall string
		if len(paramValues) > 0 {
			methodCall = fmt.Sprintf("%s.%s(%s)", serviceVar, methodName, strings.Join(paramValues, ", "))
		} else {
			methodCall = fmt.Sprintf("%s.%s()", serviceVar, methodName)
		}

		codeExample = fmt.Sprintf(`import { Client } from './client';

const client = new Client('http://localhost:9000');
const %s = client.%s();

try {
  const result = await %s;
} catch (error) {
  console.error('Error:', error);
}`, serviceVar, clientMethodName, methodCall)
	} else {
		codeExample = `import { Client } from './client';

const client = new Client('http://localhost:9000');
const service = client.jsonRpcService();

try {
  const result = await service.someMethod({ param: 'value' });
} catch (error) {
  console.error('Error:', error);
}`
	}

	md.CodeBlocks(markdown.SyntaxHighlightTypeScript, codeExample)

	md.PlainText(markdown.Bold("Коды ошибок JSON-RPC 2.0:"))
	errorCodes := []string{
		markdown.Code("-32700") + " - Parse error",
		markdown.Code("-32600") + " - Invalid Request",
		markdown.Code("-32601") + " - Method not found",
		markdown.Code("-32602") + " - Invalid params",
		markdown.Code("-32603") + " - Internal error",
		markdown.Code("-32000 to -32099") + " - Server error (зарезервировано для серверных ошибок)",
	}
	md.BulletList(errorCodes...)

	var httpContract *model.Contract
	var httpMethod *model.Method
	for _, contract := range contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
			for _, method := range contract.Methods {
				if r.methodIsHTTP(method, contract) {
					httpContract = contract
					httpMethod = method
					break
				}
			}
			if httpContract != nil {
				break
			}
		}
	}

	httpErrorsAnchor := generateAnchor("HTTP ошибки")
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", httpErrorsAnchor))
	md.LF()
	md.H3("HTTP ошибки")
	md.PlainText("При работе с HTTP клиентом ошибки обрабатываются автоматически. Если сервер возвращает HTTP статус код, отличный от ожидаемого успешного кода, клиент выбрасывает исключение.")

	var httpCodeExample string
	if httpContract != nil && httpMethod != nil {
		fileName := r.tsFileName(httpContract)
		parts := strings.Split(fileName, "_")
		serviceVar := ""
		for i, part := range parts {
			if i == 0 {
				serviceVar += part
			} else if len(part) > 0 {
				serviceVar += strings.ToUpper(string(part[0])) + part[1:]
			}
		}
		serviceVar += "HTTP"
		clientMethodName := r.getClientMethodName(httpContract)
		methodName := r.lcName(httpMethod.Name)
		args := r.argsWithoutContext(httpMethod)

		paramValues := make([]string, 0, len(args))
		for _, arg := range args {
			exampleValue := r.generateExampleValueFromVariable(arg, strings.Join(arg.Docs, "\n"), httpContract.PkgPath)
			paramValues = append(paramValues, exampleValue)
		}

		var methodCall string
		if len(paramValues) > 0 {
			methodCall = fmt.Sprintf("%s.%s(%s)", serviceVar, methodName, strings.Join(paramValues, ", "))
		} else {
			methodCall = fmt.Sprintf("%s.%s()", serviceVar, methodName)
		}

		httpCodeExample = fmt.Sprintf(`import { Client } from './client';

const client = new Client('http://localhost:9000');
const %s = client.%s();

try {
  const result = await %s;
} catch (error) {
  console.error('HTTP Error:', error);
}`, serviceVar, clientMethodName, methodCall)
	} else {
		httpCodeExample = `import { Client } from './client';

const client = new Client('http://localhost:9000');
const httpService = client.httpServiceHTTP();

try {
  const result = await httpService.getItem({ id: '123' });
} catch (error) {
  console.error('HTTP Error:', error);
}`
	}

	md.CodeBlocks(markdown.SyntaxHighlightTypeScript, httpCodeExample)

	md.PlainText("HTTP клиент автоматически проверяет статус код ответа и выбрасывает исключение, если код не соответствует ожидаемому успешному коду для метода.")
	md.LF()
	md.HorizontalRule()
}

func (r *ClientRenderer) tsTypeStringFromVariable(variable *model.Variable, pkgPath string) string {
	schema := r.walkVariable(variable.Name, pkgPath, variable, nil, false)
	return schema.typeLink()
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

func (r *ClientRenderer) generateExampleValueFromVariable(variable *model.Variable, docs, pkgPath string) string {
	if variable.IsSlice || variable.ArrayLen > 0 {
		return "[]"
	}

	if variable.MapKey != nil && variable.MapValue != nil {
		return "{}"
	}

	typeStr := r.tsTypeStringFromVariable(variable, pkgPath)
	switch typeStr {
	case "string":
		return `"example"`
	case "number", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return "0"
	case "boolean", "bool":
		return "false"
	case "Date":
		return "new Date()"
	default:
		if variable.NumberOfPointers > 0 || strings.Contains(typeStr, "| null") || strings.Contains(typeStr, "| undefined") {
			return "null"
		}
		if strings.Contains(typeStr, "{") || strings.Contains(typeStr, "Record") {
			return "{}"
		}
		return fmt.Sprintf("{} as %s", typeStr)
	}
}

func (r *ClientRenderer) getClientMethodName(contract *model.Contract) string {
	fileName := r.tsFileName(contract)
	methodName := ""
	parts := strings.Split(fileName, "_")
	for i, part := range parts {
		if i == 0 {
			methodName += part
		} else if len(part) > 0 {
			methodName += strings.ToUpper(string(part[0])) + part[1:]
		}
	}

	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
		methodName += "HTTP"
	}

	return methodName
}

func (r *ClientRenderer) getStructType(typeID, pkgPath string) (structType *model.Type, typeName string, pkg string) {
	typ, ok := r.project.Types[typeID]
	if !ok {
		return nil, "", ""
	}

	if typ.Kind != model.TypeKindStruct || typ.TypeName == "" {
		return nil, "", ""
	}

	typeName = typ.TypeName
	pkg = typ.ImportPkgPath
	if pkg == "" {
		pkg = pkgPath
	}

	return typ, typeName, pkg
}

func (r *ClientRenderer) getTypeLinkFromVariableTS(variable *model.Variable, pkgPath string) string {
	return r.getTypeLinkFromTypeRefTS(&variable.TypeRef, pkgPath)
}

func (r *ClientRenderer) getTypeLinkFromTypeRefTS(typeRef *model.TypeRef, pkgPath string) string {
	if typeRef == nil {
		return "-"
	}
	switch {
	case typeRef.IsSlice || typeRef.ArrayLen > 0:
		return r.getTypeLinkTS(typeRef.TypeID, pkgPath)
	case typeRef.MapKey != nil && typeRef.MapValue != nil:
		keyLink := r.getTypeLinkFromTypeRefTS(typeRef.MapKey, pkgPath)
		valueLink := r.getTypeLinkFromTypeRefTS(typeRef.MapValue, pkgPath)
		links := []string{}
		if keyLink != "-" {
			links = append(links, keyLink)
		}
		if valueLink != "-" {
			links = append(links, valueLink)
		}
		if len(links) > 0 {
			return strings.Join(links, ", ")
		}
		return "-"
	default:
		return r.getTypeLinkTS(typeRef.TypeID, pkgPath)
	}
}

func (r *ClientRenderer) getTypeLinkTS(typeID, pkgPath string) string {
	if r.isBuiltinType(typeID) {
		return "-"
	}

	typ, ok := r.project.Types[typeID]
	if !ok {
		return "-"
	}

	if r.isExplicitlyExcludedType(typ) {
		return "-"
	}

	if r.hasMarshaler(typ, true) || r.hasMarshaler(typ, false) {
		return "-"
	}

	if structType, _, typePkg := r.getStructType(typeID, pkgPath); structType != nil {
		typeStr := r.tsTypeString(typeID, pkgPath)

		typ, ok := r.project.Types[typeID]
		if !ok {
			return "-"
		}

		if r.isExplicitlyExcludedType(typ) {
			return "-"
		}

		if r.hasMarshaler(typ, true) || r.hasMarshaler(typ, false) {
			return "-"
		}

		namespace := ""
		switch {
		case typ.PkgName != "":
			namespace = typ.PkgName
		case typ.ImportAlias != "":
			namespace = typ.ImportAlias
		case typ.ImportPkgPath != "":
			parts := strings.Split(typ.ImportPkgPath, "/")
			if len(parts) > 0 {
				namespace = parts[len(parts)-1]
			}
		case typePkg != "":
			parts := strings.Split(typePkg, "/")
			if len(parts) > 0 {
				namespace = parts[len(parts)-1]
			}
		}

		typeNameForAnchor := typ.TypeName
		if typeNameForAnchor == "" {
			typeNameForAnchor = typeStr
		}
		fullTypeNameForAnchor := typeNameForAnchor
		if namespace != "" {
			fullTypeNameForAnchor = fmt.Sprintf("%s.%s", namespace, typeNameForAnchor)
		}

		typeAnchor := typeAnchorID(fullTypeNameForAnchor)
		if r.typeAnchorsSet != nil && !r.typeAnchorsSet[typeAnchor] {
			return "-"
		}
		return fmt.Sprintf("[%s](#%s)", typeStr, typeAnchor)
	}

	typ3, ok := r.project.Types[typeID]
	if ok && typ3.ImportPkgPath != "" && !r.isTypeFromCurrentProject(typ3.ImportPkgPath) {
		if r.isExplicitlyExcludedType(typ3) {
			return "-"
		}
		if r.hasMarshaler(typ3, true) || r.hasMarshaler(typ3, false) {
			return "-"
		}
		return "-"
	}

	return "-"
}

func (r *ClientRenderer) tsTypeString(typeID, pkgPath string) string {
	if r.isBuiltinType(typeID) {
		return r.typeIDToTSType(typeID)
	}

	typ, ok := r.project.Types[typeID]
	if !ok {
		return r.typeIDToTSType(typeID)
	}

	if r.isExplicitlyExcludedType(typ) {
		if typ.ImportPkgPath == "time" && typ.TypeName == "Time" {
			return "Date"
		}
		return r.typeIDToTSType(typeID)
	}

	if r.hasMarshaler(typ, true) || r.hasMarshaler(typ, false) {
		return "any"
	}

	typeName := typ.TypeName
	if typeName == "" {
		typeName = typeID
	}

	if typ.ImportPkgPath != "" && typ.ImportPkgPath != pkgPath {
		namespace := ""
		switch {
		case typ.PkgName != "":
			namespace = typ.PkgName
		case typ.ImportAlias != "":
			namespace = typ.ImportAlias
		default:
			parts := strings.Split(typ.ImportPkgPath, "/")
			if len(parts) > 0 {
				namespace = parts[len(parts)-1]
			}
		}
		if namespace != "" {
			return fmt.Sprintf("%s.%s", namespace, castTypeTs(typeName))
		}
	}

	return castTypeTs(typeName)
}

type typeUsageTS struct {
	typeName     string
	pkgPath      string
	fullTypeName string
	locations    []string
}

func (r *ClientRenderer) collectStructTypesTS() map[string]*typeUsageTS {
	typeUsages := make(map[string]*typeUsageTS)

	contracts := make([]*model.Contract, len(r.project.Contracts))
	copy(contracts, r.project.Contracts)
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].Name < contracts[j].Name
	})
	for _, contract := range contracts {
		for _, method := range contract.Methods {
			args := r.argsWithoutContext(method)
			for _, arg := range args {
				if structType, typeName, pkg := r.getStructType(arg.TypeID, contract.PkgPath); structType != nil {
					keyTypeName := typeName
					if typeName == "" {
						typeName = arg.Name
						keyTypeName = arg.Name
					}
					if strings.Contains(keyTypeName, ".") {
						parts := strings.Split(keyTypeName, ".")
						keyTypeName = parts[len(parts)-1]
					}
					key := fmt.Sprintf("%s.%s", pkg, keyTypeName)

					fullTypeNameForKey := r.tsTypeString(arg.TypeID, contract.PkgPath)
					if fullTypeNameForKey == "" {
						fullTypeNameForKey = typeName
					}

					if _, ok := typeUsages[key]; !ok {
						typeUsages[key] = &typeUsageTS{
							typeName:     keyTypeName,
							pkgPath:      pkg,
							fullTypeName: fullTypeNameForKey,
							locations:    make([]string, 0),
						}
					}
					location := fmt.Sprintf("%s.%s.%s", contract.Name, method.Name, arg.Name)
					typeUsages[key].locations = append(typeUsages[key].locations, location)
				}
			}

			results := r.resultsWithoutError(method)
			for _, result := range results {
				typ, ok := r.project.Types[result.TypeID]
				if ok {
					if r.isExplicitlyExcludedType(typ) {
						continue
					}
					if r.hasMarshaler(typ, false) {
						continue
					}
				}

				if structType, typeName, pkg := r.getStructType(result.TypeID, contract.PkgPath); structType != nil {
					keyTypeName := typeName
					if typeName == "" {
						typeName = result.Name
						keyTypeName = result.Name
					}
					if strings.Contains(keyTypeName, ".") {
						parts := strings.Split(keyTypeName, ".")
						keyTypeName = parts[len(parts)-1]
					}
					key := fmt.Sprintf("%s.%s", pkg, keyTypeName)

					fullTypeNameForKey := r.tsTypeString(result.TypeID, contract.PkgPath)
					if fullTypeNameForKey == "" {
						fullTypeNameForKey = typeName
					}

					if _, ok := typeUsages[key]; !ok {
						typeUsages[key] = &typeUsageTS{
							typeName:     keyTypeName,
							pkgPath:      pkg,
							fullTypeName: fullTypeNameForKey,
							locations:    make([]string, 0),
						}
					}
					location := fmt.Sprintf("%s.%s.%s", contract.Name, method.Name, result.Name)
					typeUsages[key].locations = append(typeUsages[key].locations, location)
				}
			}
		}
	}

	return typeUsages
}

func (r *ClientRenderer) renderAllTypesTS(md *markdown.Markdown, allTypes map[string]*typeUsageTS) {
	sharedTypesAnchor := generateAnchor("Общие типы")
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", sharedTypesAnchor))
	md.LF()
	md.H2("Общие типы")
	md.PlainText("Типы данных, используемые в клиенте. Типы, используемые в нескольких методах, описаны здесь для избежания дублирования.")
	md.LF()

	typesByNamespace := make(map[string][]*typeUsageTS)
	for _, usage := range common.SortedPairs(allTypes) {
		namespace := ""
		if strings.Contains(usage.fullTypeName, ".") {
			parts := strings.SplitN(usage.fullTypeName, ".", 2)
			namespace = parts[0]
		}

		if typesByNamespace[namespace] == nil {
			typesByNamespace[namespace] = make([]*typeUsageTS, 0)
		}
		typesByNamespace[namespace] = append(typesByNamespace[namespace], usage)
	}

	namespaceKeys := make([]string, 0, len(typesByNamespace))
	for ns := range typesByNamespace {
		namespaceKeys = append(namespaceKeys, ns)
	}
	sort.Strings(namespaceKeys)

	for _, namespace := range namespaceKeys {
		types := typesByNamespace[namespace]

		sort.Slice(types, func(i, j int) bool {
			return types[i].fullTypeName < types[j].fullTypeName
		})

		if namespace != "" && namespace != "time" {
			namespaceAnchor := generateAnchor(namespace)
			md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", namespaceAnchor))
			md.LF()
			md.H3(namespace)
			md.LF()
		}

		for _, usage := range types {
			var typ *model.Type
			for _, t := range common.SortedPairs(r.project.Types) {
				if t.TypeName == usage.typeName {
					tPkg := t.ImportPkgPath
					if tPkg == "" {
						tPkg = usage.pkgPath
					}
					if tPkg == usage.pkgPath {
						typ = t
						break
					}
				}
			}

			if typ != nil {
				if r.isExplicitlyExcludedType(typ) {
					continue
				}
				if r.hasMarshaler(typ, true) || r.hasMarshaler(typ, false) {
					continue
				}
			}

			isExternal := typ != nil && typ.ImportPkgPath != "" && !r.isTypeFromCurrentProject(typ.ImportPkgPath)

			// Внешние типы в TS конвертируются в встроенные — не выводим их в README.
			if typ != nil && typ.Kind == model.TypeKindStruct && !isExternal {
				r.renderStructTypeTableTS(md, typ, usage.fullTypeName, usage.pkgPath)
			} else if typ != nil && !isExternal {
				typeAnchor := typeAnchorID(usage.fullTypeName)
				md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", typeAnchor))
				md.LF()
				md.H4(usage.fullTypeName)
				md.LF()
			}
		}
	}

	md.HorizontalRule()
}

func (r *ClientRenderer) renderStructTypeTableTS(md *markdown.Markdown, structType *model.Type, typeName string, pkgPath string) {
	typeAnchor := typeAnchorID(typeName)
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", typeAnchor))
	md.LF()
	md.H4(typeName)

	rows := make([][]string, 0)
	for _, field := range structType.StructFields {
		fieldName, _ := r.jsonName(field)
		if fieldName == "-" {
			continue
		}

		typeStr := r.tsTypeStringFromStructField(field, pkgPath)

		fieldTags := parseTagsFromDocs(field.Docs)
		isRequired := fieldTags[tagRequired] != ""
		requiredStr := "Нет"
		if isRequired {
			requiredStr = "Да"
		}

		isNullable := field.NumberOfPointers > 0
		nullableStr := "Нет"
		if isNullable {
			nullableStr = "Да"
		}

		hasOmitempty := false
		if tagValues, ok := field.Tags["json"]; ok {
			for _, val := range tagValues {
				if val == "omitempty" {
					hasOmitempty = true
					break
				}
			}
		}
		omitemptyStr := "Нет"
		if hasOmitempty {
			omitemptyStr = "Да"
		}

		typeLink := r.getTypeLinkFromStructFieldTS(field, pkgPath)

		rows = append(rows, []string{
			markdown.Code(fieldName),
			markdown.Code(typeStr),
			requiredStr,
			nullableStr,
			omitemptyStr,
			typeLink,
		})
	}

	headers := []string{"Поле", "Тип", "Обязательное", "Nullable", "Omitempty", "Ссылка на тип"}

	tableSet := markdown.TableSet{
		Header: headers,
		Rows:   rows,
	}
	md.Table(tableSet)
	md.LF()
}

func (r *ClientRenderer) tsTypeStringFromStructField(field *model.StructField, pkgPath string) string {
	return r.tsTypeStringFromTypeRef(&field.TypeRef, pkgPath)
}

func (r *ClientRenderer) tsTypeStringFromTypeRef(typeRef *model.TypeRef, pkgPath string) string {
	if typeRef == nil {
		return "any"
	}
	if typeRef.IsSlice || typeRef.ArrayLen > 0 {
		elemType := r.tsTypeString(typeRef.TypeID, pkgPath)
		return fmt.Sprintf("%s[]", elemType)
	}

	if typeRef.MapKey != nil && typeRef.MapValue != nil {
		keyType := r.tsTypeStringFromTypeRef(typeRef.MapKey, pkgPath)
		valueType := r.tsTypeStringFromTypeRef(typeRef.MapValue, pkgPath)
		return fmt.Sprintf("Record<%s, %s>", keyType, valueType)
	}

	return r.tsTypeString(typeRef.TypeID, pkgPath)
}

func (r *ClientRenderer) getTypeLinkFromStructFieldTS(field *model.StructField, pkgPath string) string {
	switch {
	case field.IsSlice || field.ArrayLen > 0:
		return r.getTypeLinkTS(field.TypeID, pkgPath)
	case field.MapKey != nil && field.MapValue != nil:
		keyLink := r.getTypeLinkFromTypeRefTS(field.MapKey, pkgPath)
		valueLink := r.getTypeLinkFromTypeRefTS(field.MapValue, pkgPath)
		links := []string{}
		if keyLink != "-" {
			links = append(links, keyLink)
		}
		if valueLink != "-" {
			links = append(links, valueLink)
		}
		if len(links) > 0 {
			return strings.Join(links, ", ")
		}
		return "-"
	default:
		return r.getTypeLinkTS(field.TypeID, pkgPath)
	}
}
