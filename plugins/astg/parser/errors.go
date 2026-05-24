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

	for _, contract := range project.Contracts {
		for _, method := range contract.Methods {
			errorsFromAnnotations := extractErrorsFromAnnotations(method.Annotations)
			errorsFromImplementations := extractErrorsFromImplementations(method, contract)
			errorsFromHandlers := extractErrorsFromHandler(method, loader)

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
				enrichErrorInfoHTTPCode(errInfo, loader)
				method.Errors = append(method.Errors, errInfo)

				typeID := errInfo.TypeID
				if typeID == "" {
					continue
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

	return
}

func findErrorTypesInMethodBody(body *ast.BlockStmt, signature *types.Signature, pkgInfo *PackageInfo) (errorTypes []*model.ErrorTypeReference) {

	if body == nil || pkgInfo == nil || pkgInfo.TypeInfo == nil || pkgInfo.Types == nil {
		return
	}

	errorTypes = make([]*model.ErrorTypeReference, 0)
	seen := make(map[string]bool)

	ast.Inspect(body, func(node ast.Node) bool {

		switch n := node.(type) {
		case *ast.ReturnStmt:
			if signature == nil {
				break
			}
			results := signature.Results()
			if results == nil {
				break
			}
			for i, expr := range n.Results {
				if i >= results.Len() || !isErrorResultType(results.At(i).Type()) {
					continue
				}
				appendErrorTypeRef(expr, pkgInfo, seen, &errorTypes)
			}
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || ident.Name != "err" || i >= len(n.Rhs) {
					continue
				}
				appendErrorTypeRef(n.Rhs[i], pkgInfo, seen, &errorTypes)
			}
		}

		return true
	})

	return
}

func appendErrorTypeRef(expr ast.Expr, pkgInfo *PackageInfo, seen map[string]bool, errorTypes *[]*model.ErrorTypeReference) {

	errorRef := errorTypeRefFromExpr(expr, pkgInfo)
	if errorRef == nil {
		return
	}
	key := fmt.Sprintf("%s:%s", errorRef.PkgPath, errorRef.Name)
	if seen[key] {
		return
	}
	seen[key] = true
	*errorTypes = append(*errorTypes, errorRef)
}

func isErrorResultType(typ types.Type) (isError bool) {

	if typ == nil {
		return
	}

	typ = derefAndUnaliasType(typ)
	errorIface := createErrorInterface()
	if types.Implements(typ, errorIface) {
		isError = true
		return
	}

	isError = types.Implements(types.NewPointer(typ), errorIface)
	return
}

func errorTypeRefFromExpr(expr ast.Expr, pkgInfo *PackageInfo) (errorRef *model.ErrorTypeReference) {

	if expr == nil || pkgInfo == nil || pkgInfo.TypeInfo == nil {
		return
	}

	typ := pkgInfo.TypeInfo.TypeOf(expr)
	if typ == nil {
		return
	}

	named := namedConcreteErrorType(typ)
	if named == nil {
		return
	}

	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return
	}

	schemaPkgPath := obj.Pkg().Path()
	schemaTypeName := obj.Name()

	symbolName := symbolNameFromExpr(expr, pkgInfo)
	if symbolName == "" {
		symbolName = schemaTypeName
	}

	symbolPkgPath := symbolPkgPathFromExpr(expr, pkgInfo, schemaPkgPath)
	if symbolPkgPath == "" {
		symbolPkgPath = schemaPkgPath
	}

	errorRef = &model.ErrorTypeReference{
		PkgPath:  symbolPkgPath,
		Name:     symbolName,
		TypeName: schemaTypeName,
		FullName: fmt.Sprintf("%s.%s", symbolPkgPath, symbolName),
	}
	return
}

func symbolNameFromExpr(expr ast.Expr, pkgInfo *PackageInfo) (name string) {

	if expr == nil || pkgInfo == nil || pkgInfo.TypeInfo == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			return symbolNameFromExpr(e.X, pkgInfo)
		}
		return symbolNameFromExpr(e.X, pkgInfo)
	case *ast.SelectorExpr:
		if selection, ok := pkgInfo.TypeInfo.Selections[e]; ok && selection != nil {
			return selection.Obj().Name()
		}
		if obj, ok := pkgInfo.TypeInfo.Uses[e.Sel]; ok && obj != nil {
			return obj.Name()
		}
	case *ast.Ident:
		if obj, ok := pkgInfo.TypeInfo.Uses[e]; ok && obj != nil {
			return obj.Name()
		}
	case *ast.CompositeLit:
		return typeNameFromTypeExpr(e.Type)
	}

	return
}

func symbolPkgPathFromExpr(expr ast.Expr, pkgInfo *PackageInfo, defaultPkgPath string) (pkgPath string) {

	if expr == nil || pkgInfo == nil || pkgInfo.TypeInfo == nil {
		return defaultPkgPath
	}

	switch e := expr.(type) {
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			return symbolPkgPathFromExpr(e.X, pkgInfo, defaultPkgPath)
		}
		return symbolPkgPathFromExpr(e.X, pkgInfo, defaultPkgPath)
	case *ast.SelectorExpr:
		if selection, ok := pkgInfo.TypeInfo.Selections[e]; ok && selection != nil {
			return objectPkgPath(selection.Obj())
		}
		if obj, ok := pkgInfo.TypeInfo.Uses[e.Sel]; ok && obj != nil {
			return objectPkgPath(obj)
		}
	case *ast.Ident:
		if obj, ok := pkgInfo.TypeInfo.Uses[e]; ok && obj != nil {
			if path := objectPkgPath(obj); path != "" {
				return path
			}
		}
	case *ast.CompositeLit:
		return typePkgPathFromTypeExpr(e.Type, pkgInfo, defaultPkgPath)
	}

	return defaultPkgPath
}

func typeNameFromTypeExpr(typeExpr ast.Expr) (name string) {

	switch t := typeExpr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.StarExpr:
		return typeNameFromTypeExpr(t.X)
	}

	return
}

func typePkgPathFromTypeExpr(typeExpr ast.Expr, pkgInfo *PackageInfo, defaultPkgPath string) (pkgPath string) {

	switch t := typeExpr.(type) {
	case *ast.Ident:
		if obj, ok := pkgInfo.TypeInfo.Uses[t]; ok && obj != nil {
			if path := objectPkgPath(obj); path != "" {
				return path
			}
		}
		return defaultPkgPath
	case *ast.SelectorExpr:
		if selection, ok := pkgInfo.TypeInfo.Selections[t]; ok && selection != nil {
			return objectPkgPath(selection.Obj())
		}
	case *ast.StarExpr:
		return typePkgPathFromTypeExpr(t.X, pkgInfo, defaultPkgPath)
	}

	return defaultPkgPath
}

func objectPkgPath(obj types.Object) (pkgPath string) {

	if obj == nil || obj.Pkg() == nil {
		return
	}

	pkgPath = obj.Pkg().Path()
	return
}

func namedConcreteErrorType(typ types.Type) (named *types.Named) {

	typ = derefAndUnaliasType(typ)

	if isPredeclaredErrorType(typ) {
		return
	}

	var ok bool
	if named, ok = typ.(*types.Named); !ok {
		return
	}

	errorIface := createErrorInterface()
	if types.Implements(named, errorIface) {
		return
	}

	if types.Implements(types.NewPointer(named), errorIface) {
		return named
	}

	return
}

func isPredeclaredErrorType(typ types.Type) (isPredeclared bool) {

	errorObj := types.Universe.Lookup("error")
	if errorObj == nil {
		return
	}

	isPredeclared = types.Identical(typ, errorObj.Type())
	return
}

func derefAndUnaliasType(typ types.Type) (result types.Type) {

	result = typ
	for {
		result = types.Unalias(result)
		if pointerType, ok := result.(*types.Pointer); ok {
			result = pointerType.Elem()
			continue
		}
		break
	}
	return
}

func errorInfoFromReference(errorRef *model.ErrorTypeReference) (errInfo *model.ErrorInfo) {

	if errorRef == nil {
		return
	}

	typeID := makeTypeID(errorRef.PkgPath, errorRef.TypeName)
	if typeID == "" {
		typeID = fmt.Sprintf("%s:%s", errorRef.PkgPath, errorRef.TypeName)
	}

	errInfo = &model.ErrorInfo{
		PkgPath:  errorRef.PkgPath,
		TypeName: errorRef.Name,
		FullName: errorRef.FullName,
		TypeID:   typeID,
	}
	return
}

func errorInfosFromReferences(errorRefs []*model.ErrorTypeReference) (errors []*model.ErrorInfo) {

	errorsMap := make(map[string]*model.ErrorInfo)
	for _, errorRef := range errorRefs {
		key := fmt.Sprintf("%s:%s", errorRef.PkgPath, errorRef.Name)
		if _, exists := errorsMap[key]; exists {
			continue
		}
		errorsMap[key] = errorInfoFromReference(errorRef)
	}

	errors = make([]*model.ErrorInfo, 0, len(errorsMap))
	for _, errInfo := range errorsMap {
		errors = append(errors, errInfo)
	}
	return
}

func extractErrorsFromImplementations(method *model.Method, contract *model.Contract) (errors []*model.ErrorInfo) {

	errorsMap := make(map[string]*model.ErrorInfo)

	for _, impl := range contract.Implementations {
		implMethod, exists := impl.MethodsMap[method.Name]
		if !exists {
			continue
		}

		for _, errInfo := range errorInfosFromReferences(implMethod.ErrorTypes) {
			key := fmt.Sprintf("%s:%s", errInfo.PkgPath, errInfo.TypeName)
			errorsMap[key] = errInfo
		}
	}

	errors = make([]*model.ErrorInfo, 0, len(errorsMap))
	for _, errInfo := range errorsMap {
		errors = append(errors, errInfo)
	}

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

	return types.NewInterfaceType([]*types.Func{errorMethod}, nil).Complete()
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
	return fmt.Sprintf("HTTP %d", code)
}

func extractErrorsFromHandler(method *model.Method, loader *AutonomousPackageLoader) (errors []*model.ErrorInfo) {

	if method.Handler == nil {
		return
	}

	handlerPkgPath := method.Handler.PkgPath
	handlerName := method.Handler.Name

	var ok bool
	var pkgInfo *PackageInfo
	if pkgInfo, ok = loader.GetPackage(handlerPkgPath); !ok || pkgInfo == nil {
		var err error
		if pkgInfo, err = loader.LoadPackageLazy(handlerPkgPath); err != nil {
			slog.Debug(i18n.Msg("Failed to load handler package"),
				slog.String("package", handlerPkgPath),
				slog.String("handler", handlerName),
				slog.Any("error", err))
			return
		}
	}

	body := findFuncBodyInFiles(pkgInfo.Files, handlerName)
	if body == nil {
		return
	}

	funcDecl := findFuncDeclInFiles(pkgInfo.Files, handlerName)
	signature := funcSignatureFromDecl(funcDecl, pkgInfo.TypeInfo)

	return errorInfosFromReferences(findErrorTypesInMethodBody(body, signature, pkgInfo))
}

func findFuncDeclInFiles(files []*ast.File, funcName string) (funcDecl *ast.FuncDecl) {

	for _, file := range files {
		for _, decl := range file.Decls {
			candidate, ok := decl.(*ast.FuncDecl)
			if !ok || candidate.Name == nil || candidate.Name.Name != funcName {
				continue
			}
			funcDecl = candidate
			return
		}
	}
	return
}

func funcSignatureFromDecl(funcDecl *ast.FuncDecl, typeInfo *types.Info) (signature *types.Signature) {

	if funcDecl == nil || funcDecl.Name == nil || typeInfo == nil {
		return
	}

	obj, ok := typeInfo.Defs[funcDecl.Name]
	if !ok {
		return
	}

	fn, ok := obj.(*types.Func)
	if !ok {
		return
	}

	signature, ok = fn.Type().(*types.Signature)
	return
}

func findFuncBodyInFiles(files []*ast.File, funcName string) (body *ast.BlockStmt) {

	funcDecl := findFuncDeclInFiles(files, funcName)
	if funcDecl != nil {
		body = funcDecl.Body
	}
	return
}
