// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"sort"
	"strings"

	"tgp/internal/markdown"

	"tgp/internal/model"
)

func (r *ClientRenderer) renderMethodDoc(md *markdown.Markdown, method *model.Method, contract *model.Contract, outDir string, typeUsages map[string]*typeUsage) {

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

	r.renderMethodSignature(md, method, contract, outDir, false)

	r.renderMethodParamsAndResults(md, method, contract, typeUsages)

	r.renderMethodErrors(md, method, contract)

	md.LF()
}

func (r *ClientRenderer) renderHTTPMethodDoc(md *markdown.Markdown, method *model.Method, contract *model.Contract, outDir string, typeUsages map[string]*typeUsage) {

	httpMethod := model.GetHTTPMethod(r.project, contract, method)
	httpPath := model.GetAnnotationValue(r.project, contract, method, nil, model.TagHttpPath, "/"+ToLowerCamel(method.Name))

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
		md.PlainText(markdown.Bold("Upload (multipart):") + " тело запроса передаётся как " + markdown.Code("multipart/form-data") + " (части: " + strings.Join(parts, ", ") + "). Имя и Content-Type части — аннотации " + markdown.Code("http-part-name") + ", " + markdown.Code("http-part-content") + ".")
		md.LF()
	} else if bodyStreamArg := r.methodRequestBodyStreamArg(method); bodyStreamArg != nil {
		md.PlainText(markdown.Bold("Upload:") + " тело запроса передаётся потоком (аргумент " + markdown.Code(bodyStreamArg.Name) + " " + markdown.Code("io.Reader") + "). Content-Type из аннотации " + markdown.Code("requestContentType") + " или " + markdown.Code("application/octet-stream") + ".")
		md.LF()
	}
	if r.methodResponseMultipart(contract, method) {
		streamResults := r.methodResponseBodyStreamResults(method)
		parts := make([]string, 0, len(streamResults))
		for _, res := range streamResults {
			parts = append(parts, markdown.Code(res.Name))
		}
		md.PlainText(markdown.Bold("Download (multipart):") + " тело ответа возвращается как " + markdown.Code("multipart/form-data") + " (части: " + strings.Join(parts, ", ") + "). Вызывающий обязан закрыть все " + markdown.Code("ReadCloser") + " после чтения.")
		md.LF()
	} else if responseStreamResult := r.methodResponseBodyStreamResult(method); responseStreamResult != nil {
		md.PlainText(markdown.Bold("Download:") + " тело ответа возвращается потоком (" + markdown.Code(responseStreamResult.Name) + " " + markdown.Code("io.ReadCloser") + "). Вызывающий обязан закрыть " + markdown.Code("ReadCloser") + " после чтения.")
		md.LF()
	}

	r.renderMethodSignature(md, method, contract, outDir, true)

	r.renderMethodParamsAndResults(md, method, contract, typeUsages)

	r.renderMethodErrors(md, method, contract)

	md.LF()
}

func (r *ClientRenderer) renderMethodSignature(md *markdown.Markdown, method *model.Method, contract *model.Contract, outDir string, isHTTP bool) {
	md.PlainText(markdown.Bold("Сигнатура:"))
	md.LF()

	var sigBuilder strings.Builder
	sigBuilder.WriteString("func (s *Client")
	if isHTTP {
		sigBuilder.WriteString(contract.Name + "HTTP")
	} else {
		sigBuilder.WriteString(contract.Name)
	}
	sigBuilder.WriteString(") ")
	sigBuilder.WriteString(method.Name)
	sigBuilder.WriteString("(ctx context.Context")

	args := r.argsWithoutContext(method)
	for _, arg := range args {
		sigBuilder.WriteString(", ")
		sigBuilder.WriteString(arg.Name)
		sigBuilder.WriteString(" ")
		typeStr := r.goTypeStringFromVariable(arg, contract.PkgPath)
		sigBuilder.WriteString(typeStr)
	}

	sigBuilder.WriteString(") (")

	results := r.resultsWithoutError(method)
	if len(results) > 0 {
		for i, result := range results {
			if i > 0 {
				sigBuilder.WriteString(", ")
			}
			if result.Name != "" {
				sigBuilder.WriteString(result.Name)
				sigBuilder.WriteString(" ")
			}
			typeStr := r.goTypeStringFromVariable(result, contract.PkgPath)
			sigBuilder.WriteString(typeStr)
		}
		sigBuilder.WriteString(", ")
	}
	sigBuilder.WriteString("err error)")

	md.CodeBlocks(markdown.SyntaxHighlightGo, sigBuilder.String())
	md.LF()
}

func (r *ClientRenderer) renderMethodParamsAndResults(md *markdown.Markdown, method *model.Method, contract *model.Contract, typeUsages map[string]*typeUsage) {
	args := r.argsWithoutContext(method)
	results := r.resultsWithoutError(method)

	if len(args) > 0 {
		md.PlainText(markdown.Bold("Параметры:"))
		md.LF()

		rows := make([][]string, 0, len(args))
		for _, arg := range args {
			typeStr := r.goTypeStringFromVariable(arg, contract.PkgPath)
			argTags := r.parseTagsFromDocs(strings.Join(arg.Docs, "\n"))
			argDesc := argTags[tagDesc]

			typeLink := r.getTypeLinkFromVariable(arg, contract.PkgPath)

			rows = append(rows, []string{
				markdown.Code(arg.Name),
				markdown.Code(typeStr),
				argDesc,
				typeLink,
			})
		}

		headers := []string{"Имя", "Тип", "Описание", "Ссылка на тип"}
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
			typeStr := r.goTypeStringFromVariable(result, contract.PkgPath)
			resultTags := r.parseTagsFromDocs(strings.Join(result.Docs, "\n"))
			resultDesc := resultTags[tagDesc]

			typeLink := r.getTypeLinkFromVariable(result, contract.PkgPath)

			resultName := result.Name

			rows = append(rows, []string{
				markdown.Code(resultName),
				markdown.Code(typeStr),
				resultDesc,
				typeLink,
			})
		}

		// Ошибка
		rows = append(rows, []string{
			markdown.Code("err"),
			markdown.Code("error"),
			"Ошибка выполнения запроса",
			"-",
		})

		headers := []string{"Имя", "Тип", "Описание", "Ссылка на тип"}
		tableSet := markdown.TableSet{
			Header: headers,
			Rows:   rows,
		}
		md.Table(tableSet)
		md.LF()
	}
}

func (r *ClientRenderer) renderMethodErrors(md *markdown.Markdown, method *model.Method, contract *model.Contract) {
	if len(method.Errors) == 0 {
		return
	}

	md.LF()
	md.PlainText(markdown.Bold("Возможные ошибки:"))
	md.LF()

	errors := make([]*model.ErrorInfo, len(method.Errors))
	copy(errors, method.Errors)
	sort.Slice(errors, func(i, j int) bool {
		// Сначала ошибки с HTTP кодом, затем без
		if errors[i].HTTPCode == 0 && errors[j].HTTPCode != 0 {
			return false
		}
		if errors[i].HTTPCode != 0 && errors[j].HTTPCode == 0 {
			return true
		}
		// Если оба с кодом или оба без - сортируем по коду
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
