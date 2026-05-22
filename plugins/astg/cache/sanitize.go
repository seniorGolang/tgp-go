// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package cache

import (
	"log/slog"

	"tgp/internal/model"
)

const cacheSchemaVersion = 2

// SanitizeProject убирает nil-элементы, которые могут появиться после JSON round-trip в кэше.
func SanitizeProject(project *model.Project) (removed int) {

	if project == nil {
		return
	}

	if len(project.Contracts) > 0 {
		cleanedContracts := project.Contracts[:0]
		for _, contract := range project.Contracts {
			if contract == nil {
				removed++
				continue
			}
			cleanedContracts = append(cleanedContracts, contract)
			contract.Methods = sanitizeMethods(contract.Methods, &removed)
			contract.Implementations = compactImplementations(contract.Implementations, &removed)
		}
		project.Contracts = cleanedContracts
	}

	if len(project.Services) > 0 {
		cleanedServices := project.Services[:0]
		for _, service := range project.Services {
			if service == nil {
				removed++
				continue
			}
			cleanedServices = append(cleanedServices, service)
		}
		project.Services = cleanedServices
	}

	if len(project.Types) > 0 {
		for typeID, typ := range project.Types {
			if typ == nil {
				delete(project.Types, typeID)
				removed++
				continue
			}
			sanitizeType(typ, &removed)
		}
	}

	if removed > 0 {
		slog.Debug("sanitized cached project", slog.Int("removedNilEntries", removed))
	}

	return
}

func sanitizeMethods(methods []*model.Method, removed *int) (cleaned []*model.Method) {

	if len(methods) == 0 {
		return methods
	}

	cleaned = methods[:0]
	for _, method := range methods {
		if method == nil {
			*removed++
			continue
		}

		method.Args = compactVariables(method.Args, removed)
		method.Results = compactVariables(method.Results, removed)
		method.Errors = compactErrorInfos(method.Errors, removed)
		cleaned = append(cleaned, method)
	}

	return
}

func compactVariables(variables []*model.Variable, removed *int) (cleaned []*model.Variable) {

	if len(variables) == 0 {
		return variables
	}

	cleaned = variables[:0]
	for _, variable := range variables {
		if variable == nil {
			*removed++
			continue
		}
		sanitizeTypeRef(&variable.TypeRef, removed)
		cleaned = append(cleaned, variable)
	}

	return
}

func compactErrorInfos(errors []*model.ErrorInfo, removed *int) (cleaned []*model.ErrorInfo) {

	if len(errors) == 0 {
		return errors
	}

	cleaned = errors[:0]
	for _, errorInfo := range errors {
		if errorInfo == nil {
			*removed++
			continue
		}
		cleaned = append(cleaned, errorInfo)
	}

	return
}

func sanitizeType(typ *model.Type, removed *int) {

	typ.StructFields = compactStructFields(typ.StructFields, removed)
	typ.FunctionArgs = compactVariables(typ.FunctionArgs, removed)
	typ.FunctionResults = compactVariables(typ.FunctionResults, removed)
	typ.EmbeddedInterfaces = compactVariables(typ.EmbeddedInterfaces, removed)
	typ.InterfaceMethods = compactFunctions(typ.InterfaceMethods, removed)
	sanitizeTypeRef(typ.MapKey, removed)
	sanitizeTypeRef(typ.MapValue, removed)
}

func compactStructFields(fields []*model.StructField, removed *int) (cleaned []*model.StructField) {

	if len(fields) == 0 {
		return fields
	}

	cleaned = fields[:0]
	for _, field := range fields {
		if field == nil {
			*removed++
			continue
		}
		sanitizeTypeRef(&field.TypeRef, removed)
		cleaned = append(cleaned, field)
	}

	return
}

func compactFunctions(functions []*model.Function, removed *int) (cleaned []*model.Function) {

	if len(functions) == 0 {
		return functions
	}

	cleaned = functions[:0]
	for _, function := range functions {
		if function == nil {
			*removed++
			continue
		}
		function.Args = compactVariables(function.Args, removed)
		function.Results = compactVariables(function.Results, removed)
		cleaned = append(cleaned, function)
	}

	return
}

func sanitizeTypeRef(typeRef *model.TypeRef, removed *int) {

	if typeRef == nil {
		return
	}

	sanitizeTypeRef(typeRef.MapKey, removed)
	sanitizeTypeRef(typeRef.MapValue, removed)
}

func compactImplementations(implementations []*model.ImplementationInfo, removed *int) (cleaned []*model.ImplementationInfo) {

	if len(implementations) == 0 {
		return implementations
	}

	cleaned = implementations[:0]
	for _, impl := range implementations {
		if impl == nil {
			*removed++
			continue
		}
		if impl.MethodsMap == nil {
			cleaned = append(cleaned, impl)
			continue
		}

		for methodName, method := range impl.MethodsMap {
			if method == nil {
				delete(impl.MethodsMap, methodName)
				*removed++
				continue
			}

			cleanedErrors := method.ErrorTypes[:0]
			for _, errorRef := range method.ErrorTypes {
				if errorRef == nil {
					*removed++
					continue
				}
				cleanedErrors = append(cleanedErrors, errorRef)
			}
			method.ErrorTypes = cleanedErrors
		}

		cleaned = append(cleaned, impl)
	}

	return
}
