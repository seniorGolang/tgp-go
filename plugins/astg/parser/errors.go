// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log/slog"
	"strconv"
	"strings"

	"tgp/core/i18n"
	"tgp/internal/model"
	"tgp/internal/tags"
)

func analyzeMethodErrors(project *model.Project, loader *AutonomousPackageLoader) (err error) {

	isErrorTypeCache := make(map[string]bool)

	for _, contract := range project.Contracts {
		for _, method := range contract.Methods {
			errorsFromAnnotations := extractErrorsFromAnnotations(method.Annotations)
			errorsFromImplementations := extractErrorsFromImplementations(method, contract, project, loader, isErrorTypeCache)
			errorsFromHandlers := extractErrorsFromHandler(method, loader, isErrorTypeCache)

			errorsMap := make(map[string]*model.ErrorInfo)
			for _, errInfo := range errorsFromImplementations {
				key := fmt.Sprintf("%s:%s", errInfo.PkgPath, errInfo.TypeName)
				errorsMap[key] = errInfo
			}
			for _, errInfo := range errorsFromHandlers {
				key := fmt.Sprintf("%s:%s", errInfo.PkgPath, errInfo.TypeName)
				if _, exists := errorsMap[key]; exists {
					continue
				}
				errorsMap[key] = errInfo
			}
			for _, errInfo := range errorsFromAnnotations {
				key := fmt.Sprintf("%s:%s:%d", errInfo.PkgPath, errInfo.TypeName, errInfo.HTTPCode)
				errorsMap[key] = errInfo
			}
			method.Errors = make([]*model.ErrorInfo, 0, len(errorsMap))
			for _, errInfo := range errorsMap {
				method.Errors = append(method.Errors, errInfo)

				typeID := errInfo.TypeID
				if typeID == "" {
					typeID = fmt.Sprintf("%s:%s", errInfo.PkgPath, errInfo.TypeName)
				}
				if err = ensureTypeLoaded(typeID, project, loader); err != nil {
					slog.Debug(i18n.Msg("Error type not found, skipping"),
						slog.String("contract", contract.Name),
						slog.String("method", method.Name),
						slog.String("typeID", typeID),
						slog.String("pkgPath", errInfo.PkgPath),
						slog.String("typeName", errInfo.TypeName),
						slog.String("error", err.Error()))
				}
			}
		}
	}

	return
}

func extractErrorsFromAnnotations(methodTags tags.DocTags) (errors []*model.ErrorInfo) {

	errors = make([]*model.ErrorInfo, 0)

	for key, value := range methodTags {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(fmt.Sprintf("%v", value))

		code, err := strconv.Atoi(key)
		if err != nil {
			continue
		}

		if code < 400 || code >= 600 {
			continue
		}

		if value == "" || value == "skip" {
			continue
		}

		tokens := strings.Split(value, ":")
		if len(tokens) != 2 {
			continue
		}

		pkgPath := tokens[0]
		typeName := tokens[1]

		typeID := makeTypeID(pkgPath, typeName)
		if typeID == "" {
			typeID = fmt.Sprintf("%s:%s", pkgPath, typeName)
		}

		errInfo := &model.ErrorInfo{
			PkgPath:      pkgPath,
			TypeName:     typeName,
			FullName:     fmt.Sprintf("%s.%s", pkgPath, typeName),
			HTTPCode:     code,
			HTTPCodeText: getHTTPStatusText(code),
			TypeID:       typeID,
		}

		errors = append(errors, errInfo)
	}

	return errors
}

func extractErrorsFromImplementations(method *model.Method, contract *model.Contract, project *model.Project, loader *AutonomousPackageLoader, isErrorTypeCache map[string]bool) (errors []*model.ErrorInfo) {

	errorsMap := make(map[string]*model.ErrorInfo)

	for _, impl := range contract.Implementations {
		implMethod, exists := impl.MethodsMap[method.Name]
		if !exists {
			continue
		}

		for _, errorRef := range implMethod.ErrorTypes {
			key := fmt.Sprintf("%s:%s", errorRef.PkgPath, errorRef.TypeName)
			if _, exists := errorsMap[key]; exists {
				continue
			}

			cacheKey := fmt.Sprintf("%s:%s", errorRef.PkgPath, errorRef.TypeName)
			isError, cached := isErrorTypeCache[cacheKey]
			if !cached {
				isError = isErrorType(errorRef.PkgPath, errorRef.TypeName, loader)
				isErrorTypeCache[cacheKey] = isError
			}

			if isError {
				typeID := makeTypeID(errorRef.PkgPath, errorRef.TypeName)
				if typeID == "" {
					typeID = fmt.Sprintf("%s:%s", errorRef.PkgPath, errorRef.TypeName)
				}

				errInfo := &model.ErrorInfo{
					PkgPath:  errorRef.PkgPath,
					TypeName: errorRef.TypeName,
					FullName: errorRef.FullName,
					TypeID:   typeID,
				}

				errorsMap[key] = errInfo
			}
		}
	}

	errors = make([]*model.ErrorInfo, 0, len(errorsMap))
	for _, errInfo := range errorsMap {
		errors = append(errors, errInfo)
	}

	return
}

func isErrorType(pkgPath string, typeName string, loader *AutonomousPackageLoader) (isError bool) {

	var pkgInfo *PackageInfo
	var ok bool
	if pkgInfo, ok = loader.GetPackage(pkgPath); !ok {
		var err error
		if pkgInfo, err = loader.LoadPackageForErrorType(pkgPath, typeName); err != nil {
			slog.Debug(i18n.Msg("Failed to load package for error type check"),
				slog.String("package", pkgPath),
				slog.String("type", typeName),
				slog.Any("error", err))
			return
		}
	}

	typeObj := pkgInfo.Types.Scope().Lookup(typeName)
	if typeObj == nil {
		return
	}

	var typeNameObj *types.TypeName
	if typeNameObj, ok = typeObj.(*types.TypeName); !ok {
		return
	}

	typ := typeNameObj.Type()
	errorIface := createErrorInterface()
	implementsError := types.Implements(typ, errorIface)
	if !implementsError {
		pointerType := types.NewPointer(typ)
		implementsError = types.Implements(pointerType, errorIface)
		if !implementsError {
			return
		}
		typ = pointerType
	}

	mset := types.NewMethodSet(typ)
	codeMethod := mset.Lookup(pkgInfo.Types, "Code")
	if codeMethod == nil {
		return
	}

	var sig *types.Signature
	if sig, ok = codeMethod.Type().(*types.Signature); !ok {
		return
	}
	results := sig.Results()
	if results.Len() != 1 {
		return
	}

	resultType := results.At(0).Type()
	var basicType *types.Basic
	if basicType, ok = resultType.(*types.Basic); !ok || basicType.Kind() != types.Int {
		return
	}

	isError = true
	return
}

func createErrorInterface() (iface *types.Interface) {

	errorMethod := types.NewFunc(
		token.NoPos,
		nil,
		"Error",
		types.NewSignatureType(
			nil, // receiver
			nil, // recvTypeParams
			nil, // typeParams
			nil, // params
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", types.Typ[types.String]),
			), // results
			false, // variadic
		),
	)

	iface = types.NewInterfaceType([]*types.Func{errorMethod}, nil).Complete()
	return
}

func getHTTPStatusText(code int) (text string) {

	statusTexts := map[int]string{
		400: "Bad Request",
		401: "Unauthorized",
		403: "Forbidden",
		404: "Not Found",
		405: "Method Not Allowed",
		409: "Conflict",
		422: "Unprocessable Entity",
		429: "Too Many Requests",
		500: "Internal Server Error",
		502: "Bad Gateway",
		503: "Service Unavailable",
		504: "Gateway Timeout",
	}
	var ok bool
	if text, ok = statusTexts[code]; ok {
		return
	}
	text = fmt.Sprintf("HTTP %d", code)
	return
}

func extractErrorsFromHandler(method *model.Method, loader *AutonomousPackageLoader, isErrorTypeCache map[string]bool) (errors []*model.ErrorInfo) {

	if method.Handler == nil {
		return
	}

	handlerPkgPath := method.Handler.PkgPath
	handlerName := method.Handler.Name
	var pkgInfo *PackageInfo
	var ok bool
	var astFiles []*ast.File
	if pkgInfo, ok = loader.GetPackage(handlerPkgPath); ok && pkgInfo != nil {
		astFiles = pkgInfo.Files
	} else {
		var err error
		var fset *token.FileSet
		if astFiles, fset, err = loader.ParsePackageFilesOnly(handlerPkgPath); err != nil {
			_ = fset
			slog.Debug(i18n.Msg("Failed to parse handler package files"),
				slog.String("package", handlerPkgPath),
				slog.String("handler", handlerName),
				slog.Any("error", err))
			return nil
		}
	}

	for _, astFile := range astFiles {
		for _, decl := range astFile.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if funcDecl.Name == nil || funcDecl.Name.Name != handlerName {
				continue
			}

			if funcDecl.Body != nil {
				errorTypes := findErrorTypesInMethodBody(funcDecl.Body, astFile, handlerPkgPath, loader.resolver)
				errorsMap := make(map[string]*model.ErrorInfo)
				for _, errorRef := range errorTypes {
					key := fmt.Sprintf("%s:%s", errorRef.PkgPath, errorRef.TypeName)
					if _, exists := errorsMap[key]; exists {
						continue
					}

					cacheKey := fmt.Sprintf("%s:%s", errorRef.PkgPath, errorRef.TypeName)
					isError, cached := isErrorTypeCache[cacheKey]
					if !cached {
						isError = isErrorType(errorRef.PkgPath, errorRef.TypeName, loader)
						isErrorTypeCache[cacheKey] = isError
					}

					if !isError {
						continue
					}

					typeID := makeTypeID(errorRef.PkgPath, errorRef.TypeName)
					if typeID == "" {
						typeID = fmt.Sprintf("%s:%s", errorRef.PkgPath, errorRef.TypeName)
					}

					errInfo := &model.ErrorInfo{
						PkgPath:  errorRef.PkgPath,
						TypeName: errorRef.TypeName,
						FullName: errorRef.FullName,
						TypeID:   typeID,
					}

					errorsMap[key] = errInfo
				}

				errors = make([]*model.ErrorInfo, 0, len(errorsMap))
				for _, errInfo := range errorsMap {
					errors = append(errors, errInfo)
				}

				return
			}
		}
	}

	return
}
