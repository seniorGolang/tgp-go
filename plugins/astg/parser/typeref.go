// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/ast"
	"go/types"
	"log/slog"

	"tgp/core/i18n"
	"tgp/internal/model"
)

type typeConversionInfo struct {
	TypeID           string
	NumberOfPointers int
	IsSlice          bool
	ArrayLen         int
	IsEllipsis       bool
	ElementPointers  int
	MapKey           *model.TypeRef
	MapValue         *model.TypeRef
	ChanOf           *model.TypeRef
	ChanDirection    int
}

func typeRefFromExpr(expr ast.Expr, pkgPath string, imports map[string]string, project *model.Project, loader *AutonomousPackageLoader, typeInfo *types.Info) (info typeConversionInfo, ok bool) {

	if expr == nil {
		slog.Error(i18n.Msg("type expression is nil"), slog.String("package", pkgPath))
		return
	}
	if typeInfo == nil {
		slog.Error(i18n.Msg("types.Info is nil"), slog.String("package", pkgPath))
		return
	}

	typ := typeInfo.TypeOf(expr)
	if typ == nil {
		slog.Error(i18n.Msg("TypeOf returned nil"), slog.String("package", pkgPath), slog.String("expr", types.ExprString(expr)))
		return
	}
	if basic, isBasic := typ.(*types.Basic); isBasic && basic.Name() == "invalid type" {
		slog.Error(i18n.Msg("TypeOf returned invalid type"), slog.String("package", pkgPath), slog.String("expr", types.ExprString(expr)))
		return
	}

	if info, ok = typeRefFromTypes(typ, pkgPath, imports, project, loader); !ok {
		slog.Error(i18n.Msg("failed to convert type"), slog.String("package", pkgPath), slog.String("expr", types.ExprString(expr)), slog.String("goType", typ.String()))
		return
	}

	if _, isEllipsis := expr.(*ast.Ellipsis); isEllipsis {
		info.IsEllipsis = true
		info.IsSlice = true
	}
	return
}

func typeRefFromTypes(typ types.Type, pkgPath string, imports map[string]string, project *model.Project, loader *AutonomousPackageLoader) (info typeConversionInfo, ok bool) {

	if typ == nil {
		return
	}

	for {
		if ptr, isPtr := typ.(*types.Pointer); isPtr {
			info.NumberOfPointers++
			typ = ptr.Elem()
			continue
		}
		break
	}

	switch t := typ.(type) {
	case *types.Slice:
		info.IsSlice = true
		var eltInfo typeConversionInfo
		if eltInfo, ok = typeRefFromTypes(t.Elem(), pkgPath, imports, project, loader); !ok {
			return
		}
		// [][]byte → TypeID "[]byte" + IsSlice, чтобы отличить от []byte (TypeID "byte").
		if eltInfo.IsSlice && eltInfo.TypeID == "byte" && eltInfo.ElementPointers == 0 && !eltInfo.IsEllipsis {
			info.TypeID = "[]byte"
			return
		}
		info.TypeID = eltInfo.TypeID
		info.ElementPointers = eltInfo.NumberOfPointers
		info.MapKey = eltInfo.MapKey
		info.MapValue = eltInfo.MapValue
		info.ChanOf = eltInfo.ChanOf
		info.ChanDirection = eltInfo.ChanDirection
		return

	case *types.Array:
		info.ArrayLen = int(t.Len())
		var eltInfo typeConversionInfo
		if eltInfo, ok = typeRefFromTypes(t.Elem(), pkgPath, imports, project, loader); !ok {
			return
		}
		info.TypeID = eltInfo.TypeID
		info.ElementPointers = eltInfo.NumberOfPointers
		info.MapKey = eltInfo.MapKey
		info.MapValue = eltInfo.MapValue
		info.ChanOf = eltInfo.ChanOf
		info.ChanDirection = eltInfo.ChanDirection
		return

	case *types.Map:
		var keyInfo typeConversionInfo
		if keyInfo, ok = typeRefFromTypes(t.Key(), pkgPath, imports, project, loader); !ok {
			return
		}
		var valueInfo typeConversionInfo
		if valueInfo, ok = typeRefFromTypes(t.Elem(), pkgPath, imports, project, loader); !ok {
			return
		}
		info.MapKey = conversionInfoToTypeRef(keyInfo)
		info.MapValue = conversionInfoToTypeRef(valueInfo)
		if info.MapKey == nil || info.MapValue == nil {
			ok = false
			return
		}
		return

	case *types.Chan:
		var eltInfo typeConversionInfo
		if eltInfo, ok = typeRefFromTypes(t.Elem(), pkgPath, imports, project, loader); !ok {
			return
		}
		info.ChanOf = conversionInfoToTypeRef(eltInfo)
		if info.ChanOf == nil {
			ok = false
			return
		}
		info.ChanDirection = int(t.Dir())
		ok = true
		return

	default:
		typeID := generateTypeIDFromGoTypes(typ)
		if typeID == "" || typeID == "invalid type" {
			return
		}
		info.TypeID = typeID
		if !isBuiltinTypeName(typeID) {
			if _, err := ensureTypeInProject(typeID, typ, "", imports, project, loader); err != nil {
				slog.Error(i18n.Msg("failed to ensure type in project"), slog.String("typeID", typeID), slog.Any("error", err))
			}
		}
		ok = true
		return
	}
}

func conversionInfoToTypeRef(info typeConversionInfo) (ref *model.TypeRef) {

	if info.TypeID == "" && info.MapKey == nil && info.ChanOf == nil {
		return nil
	}
	if info.TypeID == "invalid type" {
		return nil
	}
	return &model.TypeRef{
		TypeID:           info.TypeID,
		NumberOfPointers: info.NumberOfPointers,
		IsSlice:          info.IsSlice,
		ArrayLen:         info.ArrayLen,
		IsEllipsis:       info.IsEllipsis,
		ElementPointers:  info.ElementPointers,
		MapKey:           info.MapKey,
		MapValue:         info.MapValue,
		ChanOf:           info.ChanOf,
		ChanDirection:    info.ChanDirection,
	}
}

func convertTypeFromGoTypesToInfo(typ types.Type, pkgPath string, imports map[string]string, project *model.Project, loader *AutonomousPackageLoader) (info typeConversionInfo) {

	info, _ = typeRefFromTypes(typ, pkgPath, imports, project, loader)
	return
}
