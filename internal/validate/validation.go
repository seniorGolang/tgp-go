// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package validate

import (
	"fmt"
	"strings"

	"tgp/internal/model"
)

func ValidateProject(project *model.Project) error {

	if project == nil {
		return fmt.Errorf("project cannot be nil")
	}
	if project.ModulePath == "" {
		return fmt.Errorf("project.ModulePath cannot be empty")
	}
	return nil
}

func ValidateContractID(contractID string) error {

	if contractID == "" {
		return fmt.Errorf("contractID cannot be empty")
	}
	return nil
}

func ValidateOutDir(outDir string) error {

	if outDir == "" {
		return fmt.Errorf("outDir cannot be empty")
	}
	return nil
}

func FindContract(project *model.Project, contractID string) (*model.Contract, error) {

	if project == nil {
		return nil, fmt.Errorf("project cannot be nil")
	}
	if contractID == "" {
		return nil, fmt.Errorf("contractID cannot be empty")
	}

	for _, contract := range project.Contracts {
		if contract.ID == contractID {
			return contract, nil
		}
	}

	return nil, fmt.Errorf("contract %q not found", contractID)
}

func ValidateContract(contract *model.Contract, project *model.Project) error {

	if contract == nil {
		return fmt.Errorf("contract cannot be nil")
	}

	for _, method := range contract.Methods {
		for i, arg := range method.Args {
			if arg.Name == "" && arg.TypeID != "context:Context" {
				return fmt.Errorf("contract %q: method %q: argument #%d has no name (all arguments except context.Context must be named)", contract.Name, method.Name, i+1)
			}
		}

		for i, result := range method.Results {
			if result.Name == "" && result.TypeID != "error" {
				return fmt.Errorf("contract %q: method %q: result #%d has no name (all results except error must be named)", contract.Name, method.Name, i+1)
			}
		}

		visited := make(map[string]struct{})

		for _, arg := range method.Args {
			if err := validateVariable(arg, project, contract.Name, method.Name, "argument", visited); err != nil {
				return err
			}
		}

		for _, result := range method.Results {
			if err := validateVariable(result, project, contract.Name, method.Name, "result", visited); err != nil {
				return err
			}
		}
	}

	if err := validateContractStreamTypes(contract, project); err != nil {
		return err
	}

	return nil
}

const tagHttpServer = "http-server"

const (
	typeIDIOReader     = "io:Reader"
	typeIDIOReadCloser = "io:ReadCloser"
)

func validateContractStreamTypes(contract *model.Contract, project *model.Project) error {

	hasHTTPServer := model.IsAnnotationSet(project, contract, nil, nil, tagHttpServer)

	for _, method := range contract.Methods {
		if !hasHTTPServer {
			for _, arg := range method.Args {
				if arg.TypeID == typeIDIOReader {
					return fmt.Errorf("contract %q: method %q: io.Reader в аргументах разрешён только при аннотации http-server на контракте", contract.Name, method.Name)
				}
			}
			for _, res := range method.Results {
				if res.TypeID == typeIDIOReadCloser {
					return fmt.Errorf("contract %q: method %q: io.ReadCloser в возвращаемых значениях разрешён только при аннотации http-server на контракте", contract.Name, method.Name)
				}
			}
		}
	}

	return nil
}

func validateVariable(v *model.Variable, project *model.Project, contractName, methodName, varType string, visited map[string]struct{}) error {

	if v.TypeID != "" {
		if _, seen := visited[v.TypeID]; seen {
			return nil
		}
		visited[v.TypeID] = struct{}{}
	}

	if err := validateVariableForGenerics(v, project, contractName, methodName, varType); err != nil {
		return err
	}
	if err := validateVariableForChan(v, project, contractName, methodName, varType); err != nil {
		return err
	}
	if err := validateVariableForFunction(v, project, contractName, methodName, varType); err != nil {
		return err
	}
	if err := validateVariableForUnsafe(v, project, contractName, methodName, varType); err != nil {
		return err
	}
	if err := validateVariableForInterface(v, project, contractName, methodName, varType); err != nil {
		return err
	}

	typ, ok := project.Types[v.TypeID]
	if ok && typ.Kind == model.TypeKindStruct {
		for _, field := range typ.StructFields {
			fieldVar := &model.Variable{
				TypeRef: field.TypeRef,
				Name:    field.Name,
			}
			if err := validateVariable(fieldVar, project, contractName, methodName, fmt.Sprintf("%s.%s", varType, field.Name), visited); err != nil {
				return err
			}
		}
	}

	if ok && typ.Kind == model.TypeKindAlias && typ.AliasOf != "" {
		aliasType, ok := project.Types[typ.AliasOf]
		if ok && aliasType.Kind == model.TypeKindStruct {
			for _, field := range aliasType.StructFields {
				fieldVar := &model.Variable{
					TypeRef: field.TypeRef,
					Name:    field.Name,
				}
				if err := validateVariable(fieldVar, project, contractName, methodName, fmt.Sprintf("%s.%s", varType, field.Name), visited); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func validateVariableForGenerics(v *model.Variable, project *model.Project, contractName, methodName, varType string) error {
	return validateTypeRefForGenerics(&v.TypeRef, project, contractName, methodName, varType)
}

func validateTypeRefForGenerics(typeRef *model.TypeRef, project *model.Project, contractName, methodName, varType string) error {
	if typeRef == nil {
		return nil
	}

	if strings.Contains(typeRef.TypeID, "[") && strings.Contains(typeRef.TypeID, "]") {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported generic type %q (generics are not supported)", contractName, methodName, varType, varType, typeRef.TypeID)
	}

	if typeRef.MapValue != nil {
		if err := validateTypeRefForGenerics(typeRef.MapValue, project, contractName, methodName, fmt.Sprintf("%s.mapValue", varType)); err != nil {
			return err
		}
	}

	if typeRef.MapKey != nil {
		if err := validateTypeRefForGenerics(typeRef.MapKey, project, contractName, methodName, fmt.Sprintf("%s.mapKey", varType)); err != nil {
			return err
		}
	}

	if typeRef.IsSlice {
		if strings.Contains(typeRef.TypeID, "[") && strings.Contains(typeRef.TypeID, "]") {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported generic type in slice element %q (generics are not supported)", contractName, methodName, varType, varType, typeRef.TypeID)
		}
	}

	return nil
}

func validateVariableForChan(v *model.Variable, project *model.Project, contractName, methodName, varType string) error {
	return validateTypeRefForChan(&v.TypeRef, project, contractName, methodName, varType)
}

func validateTypeRefForChan(typeRef *model.TypeRef, project *model.Project, contractName, methodName, varType string) error {
	if typeRef == nil {
		return nil
	}

	typ, ok := project.Types[typeRef.TypeID]
	if ok && typ.Kind == model.TypeKindChan {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type chan (channels are not supported)", contractName, methodName, varType, varType)
	}

	if typeRef.MapValue != nil {
		if err := validateTypeRefForChan(typeRef.MapValue, project, contractName, methodName, fmt.Sprintf("%s.mapValue", varType)); err != nil {
			return err
		}
	}

	if typeRef.MapKey != nil {
		if err := validateTypeRefForChan(typeRef.MapKey, project, contractName, methodName, fmt.Sprintf("%s.mapKey", varType)); err != nil {
			return err
		}
	}

	if typeRef.IsSlice {
		elementType, ok := project.Types[typeRef.TypeID]
		if ok && elementType.Kind == model.TypeKindChan {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type slice of chan (channels are not supported)", contractName, methodName, varType, varType)
		}
	}

	return nil
}

func validateVariableForFunction(v *model.Variable, project *model.Project, contractName, methodName, varType string) error {
	return validateTypeRefForFunction(&v.TypeRef, project, contractName, methodName, varType)
}

func validateTypeRefForFunction(typeRef *model.TypeRef, project *model.Project, contractName, methodName, varType string) error {
	if typeRef == nil {
		return nil
	}

	typ, ok := project.Types[typeRef.TypeID]
	if ok && typ.Kind == model.TypeKindFunction {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type func (function types are not supported)", contractName, methodName, varType, varType)
	}

	if typeRef.MapValue != nil {
		if err := validateTypeRefForFunction(typeRef.MapValue, project, contractName, methodName, fmt.Sprintf("%s.mapValue", varType)); err != nil {
			return err
		}
	}

	if typeRef.MapKey != nil {
		if err := validateTypeRefForFunction(typeRef.MapKey, project, contractName, methodName, fmt.Sprintf("%s.mapKey", varType)); err != nil {
			return err
		}
	}

	if typeRef.IsSlice {
		elementType, ok := project.Types[typeRef.TypeID]
		if ok && elementType.Kind == model.TypeKindFunction {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type slice of func (function types are not supported)", contractName, methodName, varType, varType)
		}
	}

	return nil
}

func validateVariableForUnsafe(v *model.Variable, project *model.Project, contractName, methodName, varType string) error {
	return validateTypeRefForUnsafe(&v.TypeRef, project, contractName, methodName, varType)
}

func validateTypeRefForUnsafe(typeRef *model.TypeRef, project *model.Project, contractName, methodName, varType string) error {
	if typeRef == nil {
		return nil
	}

	if typeRef.TypeID == "unsafe:Pointer" {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type unsafe.Pointer (unsafe types are not supported)", contractName, methodName, varType, varType)
	}

	if typeRef.MapValue != nil {
		if err := validateTypeRefForUnsafe(typeRef.MapValue, project, contractName, methodName, fmt.Sprintf("%s.mapValue", varType)); err != nil {
			return err
		}
	}

	if typeRef.MapKey != nil {
		if err := validateTypeRefForUnsafe(typeRef.MapKey, project, contractName, methodName, fmt.Sprintf("%s.mapKey", varType)); err != nil {
			return err
		}
	}

	if typeRef.IsSlice && typeRef.TypeID == "unsafe:Pointer" {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type slice of unsafe.Pointer (unsafe types are not supported)", contractName, methodName, varType, varType)
	}

	return nil
}

// validateVariableForInterface: any разрешён; именованные интерфейсы — только context.Context.
func validateVariableForInterface(v *model.Variable, project *model.Project, contractName, methodName, varType string) error {
	return validateTypeRefForInterface(&v.TypeRef, project, contractName, methodName, varType)
}

func validateTypeRefForInterface(typeRef *model.TypeRef, project *model.Project, contractName, methodName, varType string) error {
	if typeRef == nil {
		return nil
	}

	typ, ok := project.Types[typeRef.TypeID]
	if ok && typ.Kind == model.TypeKindAny {
		return nil
	}
	if ok && typ.Kind == model.TypeKindInterface {
		if typeRef.TypeID == "context:Context" {
			return nil
		}
		if typeRef.TypeID == typeIDIOReader || typeRef.TypeID == typeIDIOReadCloser {
			return nil
		}
		if strings.Contains(typeRef.TypeID, ":interface:anonymous") || typeRef.TypeID == "interface{}" {
			return nil
		}
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type interface (interfaces are not supported)", contractName, methodName, varType, varType)
	}

	if typeRef.MapValue != nil {
		if err := validateTypeRefForInterface(typeRef.MapValue, project, contractName, methodName, fmt.Sprintf("%s.mapValue", varType)); err != nil {
			return err
		}
	}

	if typeRef.MapKey != nil {
		if err := validateTypeRefForInterface(typeRef.MapKey, project, contractName, methodName, fmt.Sprintf("%s.mapKey", varType)); err != nil {
			return err
		}
	}

	if typeRef.IsSlice {
		elementType, ok := project.Types[typeRef.TypeID]
		if ok && elementType.Kind == model.TypeKindInterface {
			if strings.Contains(typeRef.TypeID, ":interface:anonymous") || typeRef.TypeID == "interface{}" {
				return nil
			}
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type slice of interface (interfaces are not supported)", contractName, methodName, varType, varType)
		}
	}

	return nil
}
