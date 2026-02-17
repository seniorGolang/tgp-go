// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package validate

import (
	"fmt"
	"strings"

	"tgp/internal/model"
)

func validateTypeRefForGenerics(typeRef *model.TypeRef, project *model.Project, contractName, methodName, varType string) (err error) {

	if typeRef == nil {
		return
	}

	if strings.Contains(typeRef.TypeID, "[") && strings.Contains(typeRef.TypeID, "]") {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported generic type %q (generics are not supported)", contractName, methodName, varType, varType, typeRef.TypeID)
	}

	if typeRef.MapValue != nil {
		if err = validateTypeRefForGenerics(typeRef.MapValue, project, contractName, methodName, fmt.Sprintf("%s.mapValue", varType)); err != nil {
			return
		}
	}

	if typeRef.MapKey != nil {
		if err = validateTypeRefForGenerics(typeRef.MapKey, project, contractName, methodName, fmt.Sprintf("%s.mapKey", varType)); err != nil {
			return
		}
	}

	if typeRef.IsSlice {
		if strings.Contains(typeRef.TypeID, "[") && strings.Contains(typeRef.TypeID, "]") {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported generic type in slice element %q (generics are not supported)", contractName, methodName, varType, varType, typeRef.TypeID)
		}
	}

	return
}

func validateTypeRefForChan(typeRef *model.TypeRef, project *model.Project, contractName, methodName, varType string) (err error) {

	if typeRef == nil {
		return
	}

	typ, ok := project.Types[typeRef.TypeID]
	if ok && typ.Kind == model.TypeKindChan {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type chan (channels are not supported)", contractName, methodName, varType, varType)
	}

	if typeRef.MapValue != nil {
		if err = validateTypeRefForChan(typeRef.MapValue, project, contractName, methodName, fmt.Sprintf("%s.mapValue", varType)); err != nil {
			return
		}
	}

	if typeRef.MapKey != nil {
		if err = validateTypeRefForChan(typeRef.MapKey, project, contractName, methodName, fmt.Sprintf("%s.mapKey", varType)); err != nil {
			return
		}
	}

	if typeRef.IsSlice {
		elementType, ok := project.Types[typeRef.TypeID]
		if ok && elementType.Kind == model.TypeKindChan {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type slice of chan (channels are not supported)", contractName, methodName, varType, varType)
		}
	}

	return
}

func validateTypeRefForFunction(typeRef *model.TypeRef, project *model.Project, contractName, methodName, varType string) (err error) {

	if typeRef == nil {
		return
	}

	typ, ok := project.Types[typeRef.TypeID]
	if ok && typ.Kind == model.TypeKindFunction {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type func (function types are not supported)", contractName, methodName, varType, varType)
	}

	if typeRef.MapValue != nil {
		if err = validateTypeRefForFunction(typeRef.MapValue, project, contractName, methodName, fmt.Sprintf("%s.mapValue", varType)); err != nil {
			return
		}
	}

	if typeRef.MapKey != nil {
		if err = validateTypeRefForFunction(typeRef.MapKey, project, contractName, methodName, fmt.Sprintf("%s.mapKey", varType)); err != nil {
			return
		}
	}

	if typeRef.IsSlice {
		elementType, ok := project.Types[typeRef.TypeID]
		if ok && elementType.Kind == model.TypeKindFunction {
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type slice of func (function types are not supported)", contractName, methodName, varType, varType)
		}
	}

	return
}

func validateTypeRefForUnsafe(typeRef *model.TypeRef, project *model.Project, contractName, methodName, varType string) (err error) {

	if typeRef == nil {
		return
	}

	if typeRef.TypeID == "unsafe:Pointer" {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type unsafe.Pointer (unsafe types are not supported)", contractName, methodName, varType, varType)
	}

	if typeRef.MapValue != nil {
		if err = validateTypeRefForUnsafe(typeRef.MapValue, project, contractName, methodName, fmt.Sprintf("%s.mapValue", varType)); err != nil {
			return
		}
	}

	if typeRef.MapKey != nil {
		if err = validateTypeRefForUnsafe(typeRef.MapKey, project, contractName, methodName, fmt.Sprintf("%s.mapKey", varType)); err != nil {
			return
		}
	}

	if typeRef.IsSlice && typeRef.TypeID == "unsafe:Pointer" {
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type slice of unsafe.Pointer (unsafe types are not supported)", contractName, methodName, varType, varType)
	}

	return
}

// any разрешён; именованные интерфейсы — только context.Context, io.Reader, io.ReadCloser и анонимные.
func validateTypeRefForInterface(typeRef *model.TypeRef, project *model.Project, contractName, methodName, varType string) (err error) {

	if typeRef == nil {
		return
	}

	var ok bool
	var typ *model.Type
	typ, ok = project.Types[typeRef.TypeID]
	if ok && typ.Kind == model.TypeKindAny {
		return
	}
	if ok && typ.Kind == model.TypeKindInterface {
		if typeRef.TypeID == "context:Context" {
			return
		}
		if typeRef.TypeID == typeIDIOReader || typeRef.TypeID == typeIDIOReadCloser {
			return
		}
		if strings.Contains(typeRef.TypeID, ":interface:anonymous") || typeRef.TypeID == "interface{}" {
			return
		}
		return fmt.Errorf("contract %q: method %q: %s %q has unsupported type interface (interfaces are not supported)", contractName, methodName, varType, varType)
	}

	if typeRef.MapValue != nil {
		if err = validateTypeRefForInterface(typeRef.MapValue, project, contractName, methodName, fmt.Sprintf("%s.mapValue", varType)); err != nil {
			return
		}
	}

	if typeRef.MapKey != nil {
		if err = validateTypeRefForInterface(typeRef.MapKey, project, contractName, methodName, fmt.Sprintf("%s.mapKey", varType)); err != nil {
			return
		}
	}

	if typeRef.IsSlice {
		var elementType *model.Type
		elementType, ok = project.Types[typeRef.TypeID]
		if ok && elementType.Kind == model.TypeKindInterface {
			if strings.Contains(typeRef.TypeID, ":interface:anonymous") || typeRef.TypeID == "interface{}" {
				return
			}
			return fmt.Errorf("contract %q: method %q: %s %q has unsupported type slice of interface (interfaces are not supported)", contractName, methodName, varType, varType)
		}
	}

	return
}
