// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"
	"sort"
	"unicode"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *transportRenderer) RenderLogViews() (err error) {

	needed := make(map[string]struct{})
	for _, contract := range r.project.Contracts {
		if !model.ContractIsHTTPFamily(r.project, contract) {
			continue
		}
		if !model.IsAnnotationSet(r.project, contract, nil, nil, TagLogger) {
			continue
		}
		cr := &contractRenderer{baseRenderer: newBaseRenderer(r.project, contract, r.outDir)}
		for _, method := range contract.Methods {
			cr.collectMethodLogProjections(method, needed)
		}
	}
	if len(needed) == 0 {
		return nil
	}

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	typeGen := types.NewGenerator(r.project, &srcFile)
	typeIDs := make([]string, 0, len(needed))
	for typeID := range needed {
		typeIDs = append(typeIDs, typeID)
	}
	sort.Strings(typeIDs)
	cr := &contractRenderer{baseRenderer: r.baseRenderer}
	for _, typeID := range typeIDs {
		cr.renderLogViewFunc(&srcFile, typeGen, typeID)
	}
	err = srcFile.Save(path.Join(r.outDir, "logview.go"))
	return
}

func (r *contractRenderer) collectMethodLogProjections(method *model.Method, needed map[string]struct{}) {

	sides, names := model.MethodLogSkip(r.project, r.contract, method)
	if !sides.Request {
		for _, arg := range streamVariables(r.project, argsWithoutContext(method), false) {
			if model.VariableLogSkipped(method, arg, names) {
				continue
			}
			r.collectTypeRefLogProjections(&arg.TypeRef, needed)
		}
	}
	if !sides.Response {
		for _, ret := range streamVariables(r.project, resultsWithoutError(method), false) {
			if model.VariableLogSkipped(method, ret, names) {
				continue
			}
			r.collectTypeRefLogProjections(&ret.TypeRef, needed)
		}
	}
}

func (r *contractRenderer) collectTypeRefLogProjections(typeRef *model.TypeRef, needed map[string]struct{}) {

	if typeRef == nil || !model.TypeRefNeedsLogProjection(r.project, typeRef) {
		return
	}
	r.collectTypeIDLogProjections(baseTypeID(typeRef), needed)
}

func (r *contractRenderer) collectTypeIDLogProjections(typeID string, needed map[string]struct{}) {

	if typeID == "" {
		return
	}
	if _, seen := needed[typeID]; seen {
		return
	}
	typ, found := r.project.Types[typeID]
	if !found || typ == nil {
		return
	}
	if typ.Kind == model.TypeKindAlias {
		r.collectTypeIDLogProjections(typ.AliasOf, needed)
		return
	}
	if typ.Kind != model.TypeKindStruct {
		if typ.IsSlice && typ.ArrayOfID != "" {
			r.collectTypeIDLogProjections(typ.ArrayOfID, needed)
		}
		return
	}
	if !model.TypeNeedsLogProjection(r.project, typeID) {
		return
	}
	needed[typeID] = struct{}{}
	for _, field := range typ.StructFields {
		if model.StructFieldLogSkipped(field) {
			continue
		}
		r.collectTypeRefLogProjections(&field.TypeRef, needed)
	}
}

func (r *contractRenderer) renderLogViewFunc(srcFile *GoFile, typeGen *types.Generator, typeID string) {

	typ, found := r.project.Types[typeID]
	if !found || typ == nil || typ.Kind != model.TypeKindStruct {
		return
	}
	funcName := r.logViewFuncName(typeID)
	srcFile.Line().Func().Id(funcName).
		Params(Id("v").Add(typeGen.FieldType(typeID, 0, false))).
		Params(Id("out").Any()).
		BlockFunc(func(bg *Group) {
			bg.Return(Map(String()).Any().Values(DictFunc(func(d Dict) {
				for _, field := range typ.StructFields {
					if model.StructFieldLogSkipped(field) {
						continue
					}
					fieldExpr := Id("v").Dot(field.Name)
					d[Lit(field.Name)] = r.logViewExpr(&field.TypeRef, fieldExpr)
				}
			})))
		})
}

func (r *contractRenderer) logViewFuncName(typeID string) (name string) {

	base := model.TypeNameFromTypeID(r.project, typeID)
	if typ, ok := r.project.Types[typeID]; ok && typ != nil && typ.ImportPkgPath != "" {
		pkg := filepath.Base(typ.ImportPkgPath)
		return "logView" + exportIdent(pkg) + base
	}
	return "logView" + base
}

func (r *contractRenderer) logViewExpr(typeRef *model.TypeRef, value Code) (expr Code) {

	if typeRef == nil || !model.TypeRefNeedsLogProjection(r.project, typeRef) {
		return value
	}
	return r.logViewWrap(typeRef, value)
}

func (r *contractRenderer) logViewWrap(typeRef *model.TypeRef, value Code) (expr Code) {

	if typeRef.NumberOfPointers > 0 {
		inner := *typeRef
		inner.NumberOfPointers--
		return r.logViewPointer(value, &inner)
	}
	if typeRef.IsSlice || typeRef.ArrayLen > 0 {
		elem := &model.TypeRef{
			TypeID:           typeRef.TypeID,
			NumberOfPointers: typeRef.ElementPointers,
			MapKey:           typeRef.MapKey,
			MapValue:         typeRef.MapValue,
		}
		return r.logViewSlice(value, elem)
	}
	if typeRef.MapValue != nil {
		return r.logViewMap(value, typeRef.MapValue)
	}
	typeID := resolveStructTypeID(r.project, typeRef.TypeID)
	if typeID == "" || !model.TypeNeedsLogProjection(r.project, typeID) {
		return value
	}
	return Id(r.logViewFuncName(typeID)).Call(value)
}

func (r *contractRenderer) logViewPointer(value Code, elem *model.TypeRef) (expr Code) {

	return Func().Params().Params(Any()).Block(
		Id("p").Op(":=").Add(value),
		If(Id("p").Op("==").Nil()).Block(Return(Nil())),
		Return(r.logViewWrap(elem, Op("*").Id("p"))),
	).Call()
}

func (r *contractRenderer) logViewSlice(value Code, elem *model.TypeRef) (expr Code) {

	return Func().Params().Params(Any()).Block(
		Id("src").Op(":=").Add(value),
		Id("out").Op(":=").Make(Index().Any(), Len(Id("src"))),
		For(List(Id("i"), Id("item")).Op(":=").Range().Id("src")).Block(
			Id("out").Index(Id("i")).Op("=").Add(r.logViewExpr(elem, Id("item"))),
		),
		Return(Id("out")),
	).Call()
}

func (r *contractRenderer) logViewMap(value Code, elem *model.TypeRef) (expr Code) {

	return Func().Params().Params(Any()).Block(
		Id("src").Op(":=").Add(value),
		Id("out").Op(":=").Make(Map(Any()).Any(), Len(Id("src"))),
		For(List(Id("k"), Id("item")).Op(":=").Range().Id("src")).Block(
			Id("out").Index(Id("k")).Op("=").Add(r.logViewExpr(elem, Id("item"))),
		),
		Return(Id("out")),
	).Call()
}

func baseTypeID(typeRef *model.TypeRef) (typeID string) {

	if typeRef == nil {
		return ""
	}
	if typeRef.TypeID != "" {
		return typeRef.TypeID
	}
	if typeRef.MapValue != nil {
		return baseTypeID(typeRef.MapValue)
	}
	if typeRef.ChanOf != nil {
		return baseTypeID(typeRef.ChanOf)
	}
	return ""
}

func resolveStructTypeID(project *model.Project, typeID string) (resolved string) {

	for typeID != "" {
		typ, found := project.Types[typeID]
		if !found || typ == nil {
			return ""
		}
		switch typ.Kind {
		case model.TypeKindStruct:
			return typeID
		case model.TypeKindAlias:
			typeID = typ.AliasOf
		default:
			return ""
		}
	}
	return ""
}

func exportIdent(s string) (out string) {

	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
