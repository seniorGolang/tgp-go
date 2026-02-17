// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package validate

import (
	"fmt"

	"tgp/internal/model"
)

func validateVariable(v *model.Variable, project *model.Project, contractName, methodName, varType string, visited map[string]struct{}) (err error) {

	if v.TypeID != "" {
		if _, seen := visited[v.TypeID]; seen {
			return
		}
		visited[v.TypeID] = struct{}{}
	}

	if err = validateTypeRefForGenerics(&v.TypeRef, project, contractName, methodName, varType); err != nil {
		return
	}
	if err = validateTypeRefForChan(&v.TypeRef, project, contractName, methodName, varType); err != nil {
		return
	}
	if err = validateTypeRefForFunction(&v.TypeRef, project, contractName, methodName, varType); err != nil {
		return
	}
	if err = validateTypeRefForUnsafe(&v.TypeRef, project, contractName, methodName, varType); err != nil {
		return
	}
	if err = validateTypeRefForInterface(&v.TypeRef, project, contractName, methodName, varType); err != nil {
		return
	}

	var ok bool
	var typ *model.Type
	typ, ok = project.Types[v.TypeID]
	if ok && typ.Kind == model.TypeKindStruct {
		for _, field := range typ.StructFields {
			fieldVar := &model.Variable{
				TypeRef: field.TypeRef,
				Name:    field.Name,
			}
			if err = validateVariable(fieldVar, project, contractName, methodName, fmt.Sprintf("%s.%s", varType, field.Name), visited); err != nil {
				return
			}
		}
	}

	if ok && typ.Kind == model.TypeKindAlias && typ.AliasOf != "" {
		var aliasType *model.Type
		aliasType, ok = project.Types[typ.AliasOf]
		if ok && aliasType.Kind == model.TypeKindStruct {
			for _, field := range aliasType.StructFields {
				fieldVar := &model.Variable{
					TypeRef: field.TypeRef,
					Name:    field.Name,
				}
				if err = validateVariable(fieldVar, project, contractName, methodName, fmt.Sprintf("%s.%s", varType, field.Name), visited); err != nil {
					return
				}
			}
		}
	}

	return
}
