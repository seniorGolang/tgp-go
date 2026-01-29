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
				Name:             field.Name,
				TypeID:           field.TypeID,
				NumberOfPointers: field.NumberOfPointers,
				IsSlice:          field.IsSlice,
				ArrayLen:         field.ArrayLen,
				IsEllipsis:       field.IsEllipsis,
				ElementPointers:  field.ElementPointers,
				MapKeyID:         field.MapKeyID,
				MapValueID:       field.MapValueID,
				MapKeyPointers:   field.MapKeyPointers,
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
					Name:             field.Name,
					TypeID:           field.TypeID,
					NumberOfPointers: field.NumberOfPointers,
					IsSlice:          field.IsSlice,
					ArrayLen:         field.ArrayLen,
					IsEllipsis:       field.IsEllipsis,
					ElementPointers:  field.ElementPointers,
					MapKeyID:         field.MapKeyID,
					MapValueID:       field.MapValueID,
					MapKeyPointers:   field.MapKeyPointers,
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

	if strings.Contains(v.TypeID, "[") && strings.Contains(v.TypeID, "]") {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported generic type %q (generics are not supported)", contractName, methodName, varType, v.Name, v.TypeID)
	}

	if v.MapValueID != "" {
		if strings.Contains(v.MapValueID, "[") && strings.Contains(v.MapValueID, "]") {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported generic type in map value %q (generics are not supported)", contractName, methodName, varType, v.Name, v.MapValueID)
		}
	}

	if v.MapKeyID != "" {
		if strings.Contains(v.MapKeyID, "[") && strings.Contains(v.MapKeyID, "]") {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported generic type in map key %q (generics are not supported)", contractName, methodName, varType, v.Name, v.MapKeyID)
		}
	}

	if v.IsSlice {
		if strings.Contains(v.TypeID, "[") && strings.Contains(v.TypeID, "]") {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported generic type in slice element %q (generics are not supported)", contractName, methodName, varType, v.Name, v.TypeID)
		}
	}

	return nil
}

func validateVariableForChan(v *model.Variable, project *model.Project, contractName, methodName, varType string) error {

	typ, ok := project.Types[v.TypeID]
	if ok && typ.Kind == model.TypeKindChan {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type chan (channels are not supported)", contractName, methodName, varType, v.Name)
	}

	if v.MapValueID != "" {
		mapValueType, ok := project.Types[v.MapValueID]
		if ok && mapValueType.Kind == model.TypeKindChan {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type map with chan value (channels are not supported)", contractName, methodName, varType, v.Name)
		}
	}

	if v.MapKeyID != "" {
		mapKeyType, ok := project.Types[v.MapKeyID]
		if ok && mapKeyType.Kind == model.TypeKindChan {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type map with chan key (channels are not supported)", contractName, methodName, varType, v.Name)
		}
	}

	if v.IsSlice {
		elementType, ok := project.Types[v.TypeID]
		if ok && elementType.Kind == model.TypeKindChan {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type slice of chan (channels are not supported)", contractName, methodName, varType, v.Name)
		}
	}

	return nil
}

func validateVariableForFunction(v *model.Variable, project *model.Project, contractName, methodName, varType string) error {

	typ, ok := project.Types[v.TypeID]
	if ok && typ.Kind == model.TypeKindFunction {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type func (function types are not supported)", contractName, methodName, varType, v.Name)
	}

	if v.MapValueID != "" {
		mapValueType, ok := project.Types[v.MapValueID]
		if ok && mapValueType.Kind == model.TypeKindFunction {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type map with func value (function types are not supported)", contractName, methodName, varType, v.Name)
		}
	}

	if v.MapKeyID != "" {
		mapKeyType, ok := project.Types[v.MapKeyID]
		if ok && mapKeyType.Kind == model.TypeKindFunction {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type map with func key (function types are not supported)", contractName, methodName, varType, v.Name)
		}
	}

	if v.IsSlice {
		elementType, ok := project.Types[v.TypeID]
		if ok && elementType.Kind == model.TypeKindFunction {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type slice of func (function types are not supported)", contractName, methodName, varType, v.Name)
		}
	}

	return nil
}

func validateVariableForUnsafe(v *model.Variable, project *model.Project, contractName, methodName, varType string) error {

	if v.TypeID == "unsafe:Pointer" {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type unsafe.Pointer (unsafe types are not supported)", contractName, methodName, varType, v.Name)
	}

	if v.MapValueID == "unsafe:Pointer" {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type map with unsafe.Pointer value (unsafe types are not supported)", contractName, methodName, varType, v.Name)
	}

	if v.MapKeyID == "unsafe:Pointer" {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type map with unsafe.Pointer key (unsafe types are not supported)", contractName, methodName, varType, v.Name)
	}

	if v.IsSlice && v.TypeID == "unsafe:Pointer" {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type slice of unsafe.Pointer (unsafe types are not supported)", contractName, methodName, varType, v.Name)
	}

	return nil
}

// validateVariableForInterface: any разрешён; именованные интерфейсы — только context.Context.
func validateVariableForInterface(v *model.Variable, project *model.Project, contractName, methodName, varType string) error {

	typ, ok := project.Types[v.TypeID]
	if ok && typ.Kind == model.TypeKindAny {
		return nil
	}
	if ok && typ.Kind == model.TypeKindInterface {
		if v.TypeID == "context:Context" {
			return nil
		}
		if strings.Contains(v.TypeID, ":interface:anonymous") || v.TypeID == "interface{}" {
			return nil
		}
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type interface (interfaces are not supported)", contractName, methodName, varType, v.Name)
	}

	if v.MapValueID != "" {
		mapValueType, ok := project.Types[v.MapValueID]
		if ok && mapValueType.Kind == model.TypeKindInterface {
			if strings.Contains(v.MapValueID, ":interface:anonymous") || v.MapValueID == "interface{}" {
				return nil
			}
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type map with interface value (interfaces are not supported)", contractName, methodName, varType, v.Name)
		}
	}

	if v.MapKeyID != "" {
		mapKeyType, ok := project.Types[v.MapKeyID]
		if ok && mapKeyType.Kind == model.TypeKindInterface {
			if strings.Contains(v.MapKeyID, ":interface:anonymous") || v.MapKeyID == "interface{}" {
				return nil
			}
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type map with interface key (interfaces are not supported)", contractName, methodName, varType, v.Name)
		}
	}

	if v.IsSlice {
		elementType, ok := project.Types[v.TypeID]
		if ok && elementType.Kind == model.TypeKindInterface {
			if strings.Contains(v.TypeID, ":interface:anonymous") || v.TypeID == "interface{}" {
				return nil
			}
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type slice of interface (interfaces are not supported)", contractName, methodName, varType, v.Name)
		}
	}

	return nil
}
