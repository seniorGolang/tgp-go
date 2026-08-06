// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"errors"
	"fmt"
	"log/slog"
	"path"

	"tgp/plugins/client-ts/tsg"

	"tgp/core/i18n"
	"tgp/internal/common"
	"tgp/internal/content"
	"tgp/internal/model"
)

func (r *ClientRenderer) RenderHTTPClientClass(contract *model.Contract) (err error) {

	if contract == nil {
		slog.Error(i18n.Msg("RenderHTTPClientClass: contract is nil"))
		return errors.New("contract is nil")
	}
	if contract.PkgPath == "" {
		slog.Error(i18n.Msg("RenderHTTPClientClass: contract.PkgPath is empty"), slog.String("contract", contract.Name))
		return fmt.Errorf("contract.PkgPath is empty for contract %s", contract.Name)
	}

	outDir := r.outDir
	r.contract = contract
	r.knownTypes = make(map[string]int)
	r.typeDefTS = make(map[string]typeDefTS)
	r.needParseFormValueHelper = false
	r.needParseCSVFormHelper = false

	file := tsg.NewFile()
	file.ImportNamed("./client", "Client")

	file.Line()

	for _, method := range contract.Methods {
		if !r.isHTTP(method, contract) {
			continue
		}
		args := r.argsWithoutContext(method)
		for _, arg := range args {
			_ = r.walkVariable(arg.Name, contract.PkgPath, arg, method.Annotations, true)
		}
		results := r.resultsWithoutError(method)
		for _, ret := range results {
			_ = r.walkVariable(ret.Name, contract.PkgPath, ret, method.Annotations, false)
		}
	}

	exchangePath := fmt.Sprintf("./%s-exchange", r.tsFileName(contract))
	exchangeTypes := make([]string, 0, len(contract.Methods)*2)
	seenTypes := make(map[string]bool)
	for _, method := range contract.Methods {
		if !r.isHTTP(method, contract) {
			continue
		}
		results := r.resultsWithoutError(method)
		if len(results) > 0 {
			if r.methodResponseMultipart(contract, method) || r.methodResponseBodyStreamResult(method) != nil {
				continue
			}
			responseType := r.responseTypeName(contract, method)
			if !seenTypes[responseType] {
				exchangeTypes = append(exchangeTypes, responseType)
				seenTypes[responseType] = true
			}
		}
	}
	if len(exchangeTypes) > 0 {
		file.ImportType(exchangePath, exchangeTypes...)
	}
	kindsUsed := make(map[string]struct{})
	for _, method := range contract.Methods {
		if !r.isHTTP(method, contract) {
			continue
		}
		kindsUsed[content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/json"))] = struct{}{}
		kindsUsed[content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json"))] = struct{}{}
	}
	for k := range kindsUsed {
		switch k {
		case content.KindXML:
			r.addNpmRuntimeDependency("fast-xml-parser")
			file.ImportNamed("fast-xml-parser", "XMLBuilder", "XMLParser")
		case content.KindMsgpack:
			r.addNpmRuntimeDependency("@msgpack/msgpack")
			file.ImportAll("@msgpack/msgpack", "Msgpack")
		case content.KindCBOR:
			r.addNpmRuntimeDependency("cbor-x")
			file.ImportAll("cbor-x", "Cbor")
		case content.KindYAML:
			r.addNpmRuntimeDependency("yaml")
			file.Import("yaml", "YAML")
		}
	}
	hasDtoTypes := false
	for _, def := range common.SortedPairs(r.typeDefTS) {
		if def.importPkg == pkgDTO {
			hasDtoTypes = true
			break
		}
	}
	if hasDtoTypes {
		file.ImportAll(exchangePath, pkgDTO)
	}

	importedTypes := make(map[string]bool)
	for _, typeName := range exchangeTypes {
		importedTypes[typeName] = true
	}

	typeByPackage := make(map[string][]typeDefTS)
	var standaloneTypes []typeDefTS

	for _, def := range common.SortedPairs(r.typeDefTS) {
		typeName := def.importName
		if typeName == "" {
			typeName = def.name
		}
		if importedTypes[typeName] {
			continue
		}
		if def.importPkg == pkgDTO {
			continue
		}

		if def.importPkg != "" {
			typeByPackage[def.importPkg] = append(typeByPackage[def.importPkg], def)
		} else {
			standaloneTypes = append(standaloneTypes, def)
		}
	}

	for _, def := range standaloneTypes {
		file.Add(r.renderTypeDef(def))
		file.Line()
	}

	for pkgName, types := range common.SortedPairs(typeByPackage) {
		file.Add(r.renderNamespace(pkgName, types))
		file.Line()
	}

	allErrorsMap := make(map[string]errorInfo)
	for _, method := range contract.Methods {
		if !r.isHTTP(method, contract) {
			continue
		}
		methodErrors := r.collectMethodErrors(method, contract)
		for key, errInfo := range methodErrors {
			if _, exists := allErrorsMap[key]; !exists {
				allErrorsMap[key] = errInfo
			}
		}
	}

	for _, errInfo := range common.SortedPairs(allErrorsMap) {
		file.Add(r.renderErrorType(errInfo))
		file.Line()
	}

	for _, method := range contract.Methods {
		if !r.isHTTP(method, contract) {
			continue
		}
		methodErrors := r.collectMethodErrors(method, contract)
		if len(methodErrors) > 0 {
			unionType := r.renderErrorUnionType(method.Name, methodErrors)
			if unionType != nil {
				file.Add(unionType)
				file.Line()
			}
		}
	}

	clientClass := r.renderHTTPClientClass(contract)
	switch {
	case r.needParseFormValueHelper && r.needParseCSVFormHelper:
		file.ImportNamed("./form", "parseFormValue", "parseCSVFormValue")
	case r.needParseCSVFormHelper:
		file.ImportNamed("./form", "parseCSVFormValue")
	case r.needParseFormValueHelper:
		file.ImportNamed("./form", "parseFormValue")
	}
	file.Add(clientClass)
	file.Line()

	file.GenerateImports()

	outFilename := path.Join(outDir, r.tsFileName(contract)+"-http.ts")
	return file.Save(outFilename)
}

func (r *ClientRenderer) renderHTTPClientClass(contract *model.Contract) (st *tsg.Statement) {
	// Захват contract в замыкании: r.contract может меняться при обходе контрактов.
	currentContract := contract

	stmt := tsg.NewStatement()
	stmt.Comment("HTTP клиент для сервиса " + contract.Name)
	stmt.Export().Class(contract.Name+"HTTPClient", func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().Private().ID("baseClient").Colon().ID("Client").Semicolon())
		grp.Line()

		constructor := tsg.NewStatement()
		constructor.Comment(fmt.Sprintf("Creates a new HTTP client for %s service", contract.Name))
		constructorStmt := tsg.NewStatement()
		constructorStmt.ID("constructor")
		constructorStmt.Params(func(cg *tsg.Group) {
			cg.Add(tsg.NewStatement().ID("_baseClient").Colon().ID("Client"))
		})
		constructorStmt.Block(func(cg *tsg.Group) {
			cg.Add(tsg.NewStatement().This().Dot("baseClient").Op("=").ID("_baseClient").Semicolon())
		})
		grp.Add(constructorStmt)
		grp.Line()

		if r.contract == nil {
			slog.Error(i18n.Msg("renderHTTPClientClass: r.contract is nil in closure"))
			return
		}
		for _, method := range currentContract.Methods {
			if !r.isHTTP(method, currentContract) {
				continue
			}
			r.renderHTTPMethod(grp, method, currentContract)
		}
	})
	stmt.Export()
	return stmt
}
