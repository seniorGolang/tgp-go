// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/constant"
	"go/types"
	"sort"

	"tgp/internal/common"
	"tgp/internal/model"
)

func attachContractEnums(project *model.Project, loader *AutonomousPackageLoader) {

	if project == nil || loader == nil || len(project.Types) == 0 {
		return
	}
	for _, typeID := range common.SortedKeys(project.Types) {
		detectEnums(project.Types[typeID], loader)
	}
}

func detectEnums(coreType *model.Type, loader *AutonomousPackageLoader) {

	if coreType == nil || loader == nil {
		return
	}
	if len(coreType.Enums) > 0 {
		return
	}
	if !isEnumEligibleType(coreType) {
		return
	}

	pkgInfo, ok := loader.GetPackage(coreType.ImportPkgPath)
	if !ok || pkgInfo == nil || pkgInfo.Types == nil {
		var err error
		if pkgInfo, err = loader.LoadPackageLazy(coreType.ImportPkgPath); err != nil || pkgInfo == nil || pkgInfo.Types == nil {
			return
		}
	}

	obj := pkgInfo.Types.Scope().Lookup(coreType.TypeName)
	if obj == nil {
		return
	}
	typeNameObj, ok := obj.(*types.TypeName)
	if !ok || typeNameObj.Type() == nil {
		return
	}
	targetType := typeNameObj.Type()

	names := pkgInfo.Types.Scope().Names()
	sort.Strings(names)

	enums := make([]*model.EnumValue, 0)
	for _, name := range names {
		obj = pkgInfo.Types.Scope().Lookup(name)
		constObj, isConst := obj.(*types.Const)
		if !isConst || constObj == nil {
			continue
		}
		if !types.Identical(constObj.Type(), targetType) {
			continue
		}
		var value string
		if value, ok = constWireValue(constObj.Val()); !ok {
			continue
		}
		enums = append(enums, &model.EnumValue{
			Name:  constObj.Name(),
			Value: value,
		})
	}

	if len(enums) < 2 {
		return
	}
	coreType.Enums = enums
}

func isEnumEligibleType(coreType *model.Type) (ok bool) {

	if coreType == nil || coreType.TypeName == "" || coreType.ImportPkgPath == "" {
		return false
	}
	if isBuiltinTypeName(coreType.TypeName) {
		return false
	}
	kind := coreType.Kind
	if coreType.UnderlyingKind != "" {
		kind = coreType.UnderlyingKind
	}
	switch kind {
	case model.TypeKindString, model.TypeKindBool,
		model.TypeKindInt, model.TypeKindInt8, model.TypeKindInt16, model.TypeKindInt32, model.TypeKindInt64,
		model.TypeKindUint, model.TypeKindUint8, model.TypeKindUint16, model.TypeKindUint32, model.TypeKindUint64,
		model.TypeKindByte, model.TypeKindRune:
		return true
	case model.TypeKindAlias:
		return coreType.UnderlyingKind != "" || coreType.UnderlyingTypeID != ""
	default:
		return false
	}
}

func constWireValue(val constant.Value) (value string, ok bool) {

	if val == nil {
		return "", false
	}
	switch val.Kind() {
	case constant.String:
		return constant.StringVal(val), true
	case constant.Int, constant.Float, constant.Bool:
		return val.ExactString(), true
	default:
		return "", false
	}
}
