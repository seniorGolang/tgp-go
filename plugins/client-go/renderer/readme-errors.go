// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path/filepath"
	"strings"

	"tgp/internal/markdown"
	"tgp/internal/model"
)

func (r *ClientRenderer) renderBatchSection(md *markdown.Markdown, contracts []*model.Contract, outDir string) {
	batchAnchor := generateAnchor("Batch запросы (JSON-RPC)")
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", batchAnchor))
	md.LF()
	md.H2("Batch запросы (JSON-RPC)")
	md.PlainText("Для выполнения нескольких JSON-RPC запросов одновременно используйте метод " + markdown.Code("Batch") + ". Это позволяет отправить несколько запросов в одном HTTP запросе.")
	md.LF()

	md.PlainText(markdown.Bold("Пример использования:"))
	r.renderBatchExample(md, contracts, outDir)
	md.LF()

	md.HorizontalRule()
}

func (r *ClientRenderer) renderErrorsSection(md *markdown.Markdown) {
	errorsAnchor := generateAnchor("Обработка ошибок")
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", errorsAnchor))
	md.LF()
	md.H2("Обработка ошибок")

	jsonrpcErrorsAnchor := generateAnchor("JSON-RPC ошибки")
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", jsonrpcErrorsAnchor))
	md.LF()
	md.H3("JSON-RPC ошибки")
	md.PlainText("При работе с JSON-RPC клиентом ошибки обрабатываются автоматически. Если сервер возвращает ошибку в формате JSON-RPC 2.0, клиент возвращает ошибку типа " + markdown.Code("error") + ".")

	pkgPath := r.pkgPath(r.outDir)
	pkgName := filepath.Base(r.outDir)
	if pkgName == "" || pkgName == "." {
		pkgName = "client"
	}

	md.CodeBlocks(markdown.SyntaxHighlightGo, fmt.Sprintf(`package main

import (
    "context"
    "fmt"
    "%s"
	"tgp/internal/model"
)

func main() {
    client := %s.New("http://localhost:9000")
    jsonRpcService := client.SomeService()

    result, err := jsonRpcService.SomeMethod(context.Background(), "example")
    if err != nil {
        fmt.Printf("Error: %%v\n", err)
        return
    }
}`, pkgPath, pkgName))

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

	httpErrorsAnchor := generateAnchor("HTTP ошибки")
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", httpErrorsAnchor))
	md.LF()
	md.H3("HTTP ошибки")
	md.PlainText("При работе с HTTP клиентом ошибки обрабатываются автоматически. Если сервер возвращает HTTP статус код, отличный от ожидаемого успешного кода, клиент возвращает ошибку типа " + markdown.Code("error") + " с описанием ошибки.")

	md.CodeBlocks(markdown.SyntaxHighlightGo, fmt.Sprintf(`package main

import (
    "context"
    "fmt"
    "%s"
	"tgp/internal/model"
)

func main() {
    client := %s.New("http://localhost:9000")
    httpService := client.SomeServiceHTTP()

    result, err := httpService.GetItem(context.Background(), "123")
    if err != nil {
        fmt.Printf("HTTP Error: %%v\n", err)
        return
    }
}`, pkgPath, pkgName))

	md.PlainText("HTTP клиент автоматически проверяет статус код ответа и возвращает ошибку, если код не соответствует ожидаемому успешному коду для метода.")

	if r.HasHTTP() {
		md.LF()
		md.PlainText(markdown.Bold("Формат тела ошибки REST.") + " Сервер отдаёт тело как " + markdown.Code("json.Encode(err)") + ", поэтому формат зависит от типа " + markdown.Code("err") + ":")
		md.LF()
		md.BulletList(
			"Строка (ошибки разбора path/query/header): тело — JSON-строка, например "+markdown.Code(`"path arguments could not be decoded: invalid id"`)+".",
			"Сгенерированный "+markdown.Code("errValidation")+" (например, "+markdown.Code("errBadRequestData(\"...\")")+"): объект "+markdown.Code(`{"trKey":"badRequest","data":"..."}`)+".",
			"Кастомные ошибки с "+markdown.Code("withErrorCode")+": формат задаётся проектом, часто "+markdown.Code(`{"code":404,"message":"not found"}`)+".",
		)
		md.LF()
		md.PlainText(markdown.Bold("Рекомендация по ErrorDecoder.") + " Дефолтный декодер ожидает формат JSON-RPC. Для REST задайте свой " + markdown.Code("ErrorDecoder") + ", парсящий фактический формат ошибок сервера (опция " + markdown.Code("WithErrorDecoder") + "). Пример типа для ошибок с полями " + markdown.Code("code") + " и " + markdown.Code("message") + ":")
		md.LF()
		md.CodeBlocks(markdown.SyntaxHighlightGo, `type restError struct {
    Code    int    `+"`json:\"code\"`"+`
    Message string `+"`json:\"message\"`"+`
}

func (e *restError) Error() string { return e.Message }`)
	}
}

func (r *ClientRenderer) renderBatchExample(md *markdown.Markdown, contracts []*model.Contract, outDir string) {
	var exampleMethods []struct {
		contract *model.Contract
		method   *model.Method
	}

	for _, contract := range contracts {
		if !model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			continue
		}
		for _, method := range contract.Methods {
			if r.methodIsJsonRPC(contract, method) {
				exampleMethods = append(exampleMethods, struct {
					contract *model.Contract
					method   *model.Method
				}{contract: contract, method: method})
				if len(exampleMethods) >= 2 {
					break
				}
			}
		}
		if len(exampleMethods) >= 2 {
			break
		}
	}

	// Если не нашли достаточно методов, используем один метод дважды или создаем минимальный пример
	pkgPath := r.pkgPath(outDir)
	pkgName := filepath.Base(outDir)
	if pkgName == "" || pkgName == "." {
		pkgName = "client"
	}

	if len(exampleMethods) == 0 {
		return
	}

	servicesUsed := make(map[string]string)
	for _, em := range exampleMethods {
		serviceVar := ToLowerCamel(em.contract.Name)
		if _, exists := servicesUsed[em.contract.Name]; !exists {
			servicesUsed[em.contract.Name] = serviceVar
		}
	}

	type ServiceInfo struct {
		Name string
		Var  string
	}
	type CallbackInfo struct {
		Name         string
		ContractName string
		MethodName   string
		ResultVar    string
		ResultType   string
		HasResult    bool
		NeedsDto     bool
		Results      []struct {
			Name string
			Type string
		}
		HasError bool
	}
	type RequestInfo struct {
		ServiceVar   string
		MethodName   string
		CallbackName string
		Params       string
		HasParams    bool
	}

	services := make([]ServiceInfo, 0, len(servicesUsed))
	for contractName, serviceVar := range servicesUsed {
		services = append(services, ServiceInfo{
			Name: contractName,
			Var:  serviceVar,
		})
	}

	callbacks := make([]CallbackInfo, 0, len(exampleMethods))
	requests := make([]RequestInfo, 0, len(exampleMethods))

	for i, em := range exampleMethods {
		serviceVar := ToLowerCamel(em.contract.Name)
		callbackName := fmt.Sprintf("callback%d", i+1)
		resultsWithoutErr := r.resultsWithoutError(em.method)

		var callbackInfo CallbackInfo
		callbackInfo.Name = callbackName
		callbackInfo.ContractName = em.contract.Name
		callbackInfo.MethodName = em.method.Name

		callbackInfo.HasError = r.isErrorLast(em.method.Results)

		allResults := em.method.Results
		callbackInfo.Results = make([]struct {
			Name string
			Type string
		}, 0, len(allResults))

		// Сначала добавляем все результаты без error
		for j, result := range resultsWithoutErr {
			var resultName string
			if result.Name != "" {
				resultName = ToLowerCamel(result.Name)
			} else {
				// Если несколько результатов без имен, используем индексы
				if len(resultsWithoutErr) == 1 {
					resultName = fmt.Sprintf("result%d", i+1)
				} else {
					resultName = fmt.Sprintf("result%d_%d", i+1, j+1)
				}
			}

			resultTypeStr := r.goTypeStringFromVariable(result, em.contract.PkgPath)
			callbackInfo.Results = append(callbackInfo.Results, struct {
				Name string
				Type string
			}{
				Name: resultName,
				Type: resultTypeStr,
			})

			if !callbackInfo.NeedsDto {
				if typ, ok := r.project.Types[result.TypeID]; ok && typ.ImportPkgPath != "" {
					if r.isTypeFromCurrentProject(typ.ImportPkgPath) {
						callbackInfo.NeedsDto = true
					}
				}
			}
		}

		// Затем добавляем error в конце (если есть)
		if callbackInfo.HasError && len(allResults) > 0 {
			lastResult := allResults[len(allResults)-1]
			resultTypeStr := r.goTypeStringFromVariable(lastResult, em.contract.PkgPath)
			callbackInfo.Results = append(callbackInfo.Results, struct {
				Name string
				Type string
			}{
				Name: "err",
				Type: resultTypeStr,
			})
		}

		if len(resultsWithoutErr) > 0 {
			callbackInfo.HasResult = true
			if len(callbackInfo.Results) > 0 && callbackInfo.Results[0].Name != "" {
				callbackInfo.ResultVar = callbackInfo.Results[0].Name
			} else {
				callbackInfo.ResultVar = fmt.Sprintf("result%d", i+1)
			}
			callbackInfo.ResultType = r.goTypeStringFromVariable(resultsWithoutErr[0], em.contract.PkgPath)
		}

		callbacks = append(callbacks, callbackInfo)

		// Подготавливаем параметры для запроса
		args := r.argsWithoutContext(em.method)
		var paramValues []string
		for _, arg := range args {
			exampleValue := r.generateExampleValueFromVariable(arg, strings.Join(arg.Docs, "\n"), em.contract.PkgPath)
			paramValues = append(paramValues, exampleValue)
		}

		requestInfo := RequestInfo{
			ServiceVar:   serviceVar,
			MethodName:   em.method.Name,
			CallbackName: callbackName,
			HasParams:    len(paramValues) > 0,
		}
		if len(paramValues) > 0 {
			requestInfo.Params = strings.Join(paramValues, ", ")
		}
		requests = append(requests, requestInfo)
	}

	needsDto := false
	for _, cb := range callbacks {
		if cb.NeedsDto {
			needsDto = true
			break
		}
	}

	templateData := map[string]any{
		"PkgPath":   pkgPath,
		"PkgName":   pkgName,
		"Services":  services,
		"Callbacks": callbacks,
		"Requests":  requests,
		"NeedsDto":  needsDto,
	}

	batchExample, err := r.renderTemplate("templates/batch_example.tmpl", templateData)
	if err != nil {
		batchExample = fmt.Sprintf("// Error rendering template: %v", err)
	}
	md.CodeBlocks(markdown.SyntaxHighlightGo, batchExample)
}
