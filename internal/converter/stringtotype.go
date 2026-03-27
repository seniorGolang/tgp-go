// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package converter

import (
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

const (
	pkgStrconv = "strconv"
	pkgStrings = "strings"
	pkgTime    = "time"
)

type FieldTypeFunc func(typeID string, numberOfPointers int, allowEllipsis bool) *Statement

// StringToTypeConfig задаёт параметры генерации кода конвертации строки в тип.
type StringToTypeConfig struct {
	Project        *model.Project
	From           *Statement
	Arg            *model.Variable
	Id             *Statement
	ErrBody        []Code
	OptionalAssign bool
	FieldType      FieldTypeFunc
	AddImport      func(pkgPath string, name string)
	JSONPkg        string

	// Для OptionalAssign: запись в родительский Group без блока, переиспользование err.
	AddTo      *Group
	ErrVar     *Statement
	ValVarName string
}

// BuildStringToType генерирует код присваивания id из строкового выражения from с учётом типа arg.
// Если OptionalAssign == true, присваивание выполняется только при err == nil (режим клиента).
// Иначе после парсинга добавляется ErrBody в блок if err != nil (режим сервера).
func BuildStringToType(cfg StringToTypeConfig) (st *Statement) {

	if cfg.Arg.IsSlice {
		return buildStringToTypeSlice(cfg)
	}
	if IsBuiltinTypeID(cfg.Arg.TypeID) {
		return buildStringToTypeBuiltin(cfg)
	}
	typ, ok := cfg.Project.Types[cfg.Arg.TypeID]
	if !ok {
		return cfg.Id.Op("=").Add(cfg.From)
	}
	if strings.Contains(typ.ImportPkgPath, "time") {
		if typ.TypeName == "Time" {
			return buildStringToTypeTime(cfg)
		}
	}
	if typ.ParseFromString != nil {
		return buildStringToTypeParseFromString(cfg, typ)
	}
	var builtinBaseTypeID string
	var hasBuiltinBase bool
	if builtinBaseTypeID, hasBuiltinBase = resolveBuiltinBaseTypeID(cfg.Project, cfg.Arg.TypeID); hasBuiltinBase && !IsBuiltinTypeID(cfg.Arg.TypeID) {
		return buildStringToTypeNamedBuiltinLenient(cfg, builtinBaseTypeID)
	}
	return buildStringToTypeJSON(cfg)
}

func resolveBuiltinBaseTypeID(project *model.Project, typeID string) (baseTypeID string, ok bool) {

	visited := make(map[string]struct{})
	return resolveBuiltinBaseTypeIDRecursive(project, typeID, visited)
}

func HasBuiltinScalarBase(project *model.Project, typeID string) (ok bool) {

	_, ok = resolveBuiltinBaseTypeID(project, typeID)
	return
}

func resolveBuiltinBaseTypeIDRecursive(project *model.Project, typeID string, visited map[string]struct{}) (baseTypeID string, ok bool) {

	if IsBuiltinTypeID(typeID) {
		baseTypeID = typeID
		ok = true
		return
	}
	if project == nil {
		return
	}
	if _, exists := visited[typeID]; exists {
		return
	}
	visited[typeID] = struct{}{}
	var typ *model.Type
	var exists bool
	if typ, exists = project.Types[typeID]; !exists || typ == nil {
		return
	}
	if typ.UnderlyingTypeID != "" && IsBuiltinTypeID(typ.UnderlyingTypeID) {
		baseTypeID = typ.UnderlyingTypeID
		ok = true
		return
	}
	if typ.UnderlyingKind != "" && IsBuiltinTypeID(string(typ.UnderlyingKind)) {
		baseTypeID = string(typ.UnderlyingKind)
		ok = true
		return
	}
	if typ.Kind != "" && IsBuiltinTypeID(string(typ.Kind)) {
		baseTypeID = string(typ.Kind)
		ok = true
		return
	}
	if typ.AliasOf != "" {
		return resolveBuiltinBaseTypeIDRecursive(project, typ.AliasOf, visited)
	}
	return
}

func buildStringToTypeSlice(cfg StringToTypeConfig) (st *Statement) {

	elementTypeID := cfg.Arg.TypeID
	elementVar := &model.Variable{
		TypeRef: model.TypeRef{
			TypeID:           elementTypeID,
			NumberOfPointers: cfg.Arg.ElementPointers,
			IsSlice:          false,
		},
		Name: "elem",
	}
	cfg.AddImport(pkgStrings, "strings")
	elementTypeCode := cfg.FieldType(elementTypeID, cfg.Arg.ElementPointers, false)
	elementCfg := cfg
	elementCfg.Arg = elementVar
	elementCfg.Id = Id("elem")
	return BlockFunc(func(bg *Group) {
		bg.Id("parts").Op(":=").Qual(pkgStrings, "Split").Call(cfg.From, Lit(","))
		bg.Id("result").Op(":=").Make(Index().Add(elementTypeCode), Lit(0), Len(Id("parts")))
		bg.For(List(Id("_"), Id("elemStr")).Op(":=").Range().Id("parts")).BlockFunc(func(ig *Group) {
			ig.Id("elemStr").Op("=").Qual(pkgStrings, "TrimSpace").Call(Id("elemStr"))
			ig.If(Id("elemStr").Op("==").Lit("")).Block(Continue())
			ig.Var().Id("elem").Add(elementTypeCode)
			ig.Add(BuildStringToType(elementCfg))
			ig.Id("result").Op("=").Append(Id("result"), Id("elem"))
		})
		bg.Add(cfg.Id).Op("=").Id("result")
	})
}

func buildStringToTypeBuiltin(cfg StringToTypeConfig) (st *Statement) {

	op := "="
	cfg.AddImport(pkgStrconv, "strconv")

	switch cfg.Arg.TypeID {
	case "string":
		return cfg.Id.Op(op).Add(cfg.From)
	case "bool":
		return assignBuiltin(cfg, Qual(pkgStrconv, "ParseBool").Call(cfg.From))
	case "int":
		return assignBuiltin(cfg, Qual(pkgStrconv, "Atoi").Call(cfg.From))
	case "int64":
		return assignBuiltin(cfg, Qual(pkgStrconv, "ParseInt").Call(cfg.From, Lit(10), Lit(64)))
	case "int32":
		return assignBuiltin(cfg, Qual(pkgStrconv, "ParseInt").Call(cfg.From, Lit(10), Lit(32)))
	case "uint":
		return assignBuiltin(cfg, Qual(pkgStrconv, "ParseUint").Call(cfg.From, Lit(10), Lit(64)))
	case "uint64":
		return assignBuiltin(cfg, Qual(pkgStrconv, "ParseUint").Call(cfg.From, Lit(10), Lit(64)))
	case "uint32":
		return assignBuiltin(cfg, Qual(pkgStrconv, "ParseUint").Call(cfg.From, Lit(10), Lit(32)))
	case "float64":
		return assignBuiltin(cfg, Qual(pkgStrconv, "ParseFloat").Call(cfg.From, Lit(64)))
	case "float32":
		parseCallF32 := Qual(pkgStrconv, "ParseFloat").Call(cfg.From, Lit(32))
		if cfg.OptionalAssign && cfg.AddTo != nil && cfg.ErrVar != nil && cfg.ValVarName != "" {
			cfg.AddTo.Var().Id(cfg.ValVarName).Float64()
			if cfg.Arg.NumberOfPointers > 0 {
				cfg.AddTo.Var().Id(cfg.ValVarName + "V").Float32()
				cfg.AddTo.If(List(Id(cfg.ValVarName), cfg.ErrVar).Op("=").Add(parseCallF32).Op(";").Add(cfg.ErrVar).Op("==").Nil()).Block(
					Id(cfg.ValVarName+"V").Op("=").Float32().Call(Id(cfg.ValVarName)),
					addAssign(cfg.Id, Id(cfg.ValVarName+"V"), 1),
				)
			} else {
				cfg.AddTo.If(List(Id(cfg.ValVarName), cfg.ErrVar).Op("=").Add(parseCallF32).Op(";").Add(cfg.ErrVar).Op("==").Nil()).Block(
					addAssign(cfg.Id, Float32().Call(Id(cfg.ValVarName)), 0),
				)
			}
			return nil
		}
		if cfg.OptionalAssign {
			return BlockFunc(func(bg *Group) {
				bg.Var().Id("temp64").Float64()
				bg.Var().Id("_e").Error()
				bg.If(List(Id("temp64"), Id("_e")).Op("=").Add(parseCallF32).Op(";").Id("_e").Op("==").Nil()).Block(
					addAssign(cfg.Id, Float32().Call(Id("temp64")), cfg.Arg.NumberOfPointers),
				)
			})
		}
		return BlockFunc(func(bg *Group) {
			bg.Var().Id("temp64").Float64()
			bg.Add(serverIfErr(Id("temp64"), parseCallF32, cfg.ErrBody))
			bg.Add(cfg.Id.Op(op).Float32().Call(Id("temp64")))
		})
	default:
		return cfg.Id.Op(op).Add(cfg.From)
	}
}

func assignBuiltin(cfg StringToTypeConfig, parseCall *Statement) (st *Statement) {

	if cfg.OptionalAssign && optionalWriteToGroup(cfg, parseCall) {
		return nil
	}
	if cfg.OptionalAssign {
		return BlockFunc(func(bg *Group) {
			bg.Var().Id("_v").Add(cfg.FieldType(cfg.Arg.TypeID, 0, false))
			bg.Var().Id("_e").Error()
			bg.If(List(Id("_v"), Id("_e")).Op("=").Add(parseCall).Op(";").Id("_e").Op("==").Nil()).Block(
				addAssign(cfg.Id, Id("_v"), cfg.Arg.NumberOfPointers),
			)
		})
	}
	return serverIfErr(cfg.Id, parseCall, cfg.ErrBody)
}

func addAssign(id *Statement, value Code, numberOfPointers int) (c Code) {

	if numberOfPointers > 0 {
		return id.Op("=").Op("&").Add(value)
	}
	return id.Op("=").Add(value)
}

func serverIfErr(id *Statement, parseCall Code, errBody []Code) (st *Statement) {

	if len(errBody) == 0 {
		return List(id, Err()).Op("=").Add(parseCall)
	}
	return If(List(id, Err()).Op("=").Add(parseCall).Op(";").Err().Op("!=").Nil()).Block(errBody...)
}

func optionalWriteToGroup(cfg StringToTypeConfig, parseCall *Statement) (ok bool) {

	if cfg.AddTo == nil || cfg.ErrVar == nil {
		return false
	}
	ptr := cfg.Arg.NumberOfPointers > 0
	if ptr && cfg.ValVarName == "" {
		return false
	}
	if ptr {
		cfg.AddTo.Var().Id(cfg.ValVarName).Add(cfg.FieldType(cfg.Arg.TypeID, 0, false))
		cfg.AddTo.If(List(Id(cfg.ValVarName), cfg.ErrVar).Op("=").Add(parseCall).Op(";").Add(cfg.ErrVar).Op("==").Nil()).Block(
			addAssign(cfg.Id, Id(cfg.ValVarName), 1),
		)
	} else {
		cfg.AddTo.If(List(cfg.Id, cfg.ErrVar).Op("=").Add(parseCall).Op(";").Add(cfg.ErrVar).Op("==").Nil()).Block()
	}
	return true
}

func buildStringToTypeTime(cfg StringToTypeConfig) (st *Statement) {

	cfg.AddImport(pkgTime, "time")
	parseCall := Qual(pkgTime, "Parse").Call(Qual(pkgTime, "RFC3339"), cfg.From)
	if cfg.OptionalAssign && optionalWriteToGroup(cfg, parseCall) {
		return nil
	}
	if cfg.OptionalAssign {
		return BlockFunc(func(bg *Group) {
			bg.Var().Id("_v").Qual(pkgTime, "Time")
			bg.Var().Id("_e").Error()
			bg.If(List(Id("_v"), Id("_e")).Op("=").Add(parseCall).Op(";").Id("_e").Op("==").Nil()).Block(
				addAssign(cfg.Id, Id("_v"), cfg.Arg.NumberOfPointers),
			)
		})
	}
	return serverIfErr(cfg.Id, parseCall, cfg.ErrBody)
}

func buildStringToTypeParseFromString(cfg StringToTypeConfig, typ *model.Type) (st *Statement) {

	cfg.AddImport(typ.ImportPkgPath, filepath.Base(typ.ImportPkgPath))
	call := Qual(typ.ImportPkgPath, typ.ParseFromString.FuncName).Call(cfg.From)
	fieldPtr := cfg.Arg.NumberOfPointers > 0
	returnsPtr := typ.ParseFromString.ReturnsPointer
	returnsErr := typ.ParseFromString.ReturnsError
	op := "="

	if cfg.OptionalAssign {
		return buildParseFromStringOptional(cfg, call, fieldPtr, returnsPtr, returnsErr)
	}

	if !fieldPtr && !returnsPtr {
		if returnsErr {
			return serverIfErr(cfg.Id, call, cfg.ErrBody)
		}
		return List(cfg.Id, Id("_")).Op(op).Add(call)
	}
	if !fieldPtr && returnsPtr {
		return BlockFunc(func(bg *Group) {
			bg.Var().Id("_parseTmp").Add(cfg.FieldType(cfg.Arg.TypeID, 1, false))
			if returnsErr {
				bg.Add(serverIfErr(Id("_parseTmp"), call, cfg.ErrBody))
			} else {
				bg.List(Id("_parseTmp"), Id("_")).Op(op).Add(call)
			}
			bg.Add(cfg.Id.Op(op).Op("*").Id("_parseTmp"))
		})
	}
	if fieldPtr && returnsPtr {
		if returnsErr {
			return serverIfErr(cfg.Id, call, cfg.ErrBody)
		}
		return List(cfg.Id, Id("_")).Op(op).Add(call)
	}
	return BlockFunc(func(bg *Group) {
		bg.Var().Id("_parseTmp").Add(cfg.FieldType(cfg.Arg.TypeID, 0, false))
		if returnsErr {
			bg.Add(serverIfErr(Id("_parseTmp"), call, cfg.ErrBody))
		} else {
			bg.List(Id("_parseTmp"), Id("_")).Op(op).Add(call)
		}
		bg.Add(cfg.Id.Op(op).Op("&").Id("_parseTmp"))
	})
}

func buildParseFromStringOptional(cfg StringToTypeConfig, call *Statement, fieldPtr bool, returnsPtr bool, returnsErr bool) (st *Statement) {

	if cfg.AddTo != nil && cfg.ErrVar != nil && cfg.ValVarName != "" {
		writeParseFromStringToGroup(cfg, call, fieldPtr, returnsPtr, returnsErr)
		return nil
	}

	valVar := Id("_parseVal")
	errVar := Id("_parseErr")

	if !fieldPtr && !returnsPtr {
		if returnsErr {
			return BlockFunc(func(bg *Group) {
				bg.Var().Id("_parseVal").Add(cfg.FieldType(cfg.Arg.TypeID, 0, false))
				bg.Var().Id("_parseErr").Error()
				bg.If(List(valVar, errVar).Op("=").Add(call).Op(";").Id("_parseErr").Op("==").Nil()).Block(
					addAssign(cfg.Id, valVar, 0),
				)
			})
		}
		return BlockFunc(func(bg *Group) {
			bg.List(valVar, Id("_")).Op("=").Add(call)
			bg.Add(addAssign(cfg.Id, valVar, 0))
		})
	}
	if fieldPtr && returnsPtr {
		if returnsErr {
			return BlockFunc(func(bg *Group) {
				bg.Var().Id("_parseVal").Add(cfg.FieldType(cfg.Arg.TypeID, 1, false))
				bg.Var().Id("_parseErr").Error()
				bg.If(List(valVar, errVar).Op("=").Add(call).Op(";").Id("_parseErr").Op("==").Nil()).Block(
					addAssign(cfg.Id, valVar, 1),
				)
			})
		}
		return BlockFunc(func(bg *Group) {
			bg.List(valVar, Id("_")).Op("=").Add(call)
			bg.Add(addAssign(cfg.Id, valVar, 1))
		})
	}
	if !fieldPtr && returnsPtr {
		return BlockFunc(func(bg *Group) {
			bg.Var().Id("_parseVal").Add(cfg.FieldType(cfg.Arg.TypeID, 1, false))
			bg.Var().Id("_parseErr").Error()
			bg.If(List(valVar, errVar).Op("=").Add(call).Op(";").Id("_parseErr").Op("==").Nil()).Block(
				addAssign(cfg.Id, Op("*").Add(valVar), 0),
			)
		})
	}
	return BlockFunc(func(bg *Group) {
		bg.Var().Id("_parseVal").Add(cfg.FieldType(cfg.Arg.TypeID, 0, false))
		bg.Var().Id("_parseErr").Error()
		bg.If(List(valVar, errVar).Op("=").Add(call).Op(";").Id("_parseErr").Op("==").Nil()).Block(
			addAssign(cfg.Id, Op("&").Add(valVar), 1),
		)
	})
}

func writeParseFromStringToGroup(cfg StringToTypeConfig, call *Statement, fieldPtr bool, returnsPtr bool, returnsErr bool) {

	g := cfg.AddTo
	v := Id(cfg.ValVarName)
	ev := cfg.ErrVar

	if !fieldPtr && !returnsPtr {
		if returnsErr {
			g.If(List(cfg.Id, ev).Op("=").Add(call).Op(";").Add(ev).Op("==").Nil()).Block()
		} else {
			g.List(cfg.Id, Id("_")).Op("=").Add(call)
		}
		return
	}
	if fieldPtr && returnsPtr {
		g.Var().Id(cfg.ValVarName).Add(cfg.FieldType(cfg.Arg.TypeID, 1, false))
		if returnsErr {
			g.If(List(v, ev).Op("=").Add(call).Op(";").Add(ev).Op("==").Nil()).Block(
				addAssign(cfg.Id, v, 1),
			)
		} else {
			g.List(v, Id("_")).Op("=").Add(call)
			g.Add(addAssign(cfg.Id, v, 1))
		}
		return
	}
	if !fieldPtr && returnsPtr {
		g.Var().Id(cfg.ValVarName).Add(cfg.FieldType(cfg.Arg.TypeID, 1, false))
		g.If(List(v, ev).Op("=").Add(call).Op(";").Add(ev).Op("==").Nil()).Block(
			addAssign(cfg.Id, Op("*").Add(v), 0),
		)
		return
	}
	g.Var().Id(cfg.ValVarName).Add(cfg.FieldType(cfg.Arg.TypeID, 0, false))
	g.If(List(v, ev).Op("=").Add(call).Op(";").Add(ev).Op("==").Nil()).Block(
		addAssign(cfg.Id, v, 1),
	)
}

func buildStringToTypeJSON(cfg StringToTypeConfig) (st *Statement) {

	cfg.AddImport(cfg.JSONPkg, "json")
	return Op("_").Op("=").Qual(cfg.JSONPkg, "Unmarshal").Call(Op("[]").Byte().Call(Op("`\"`").Op("+").Add(cfg.From).Op("+").Op("`\"`")), Op("&").Add(cfg.Id))
}

func buildStringToTypeNamedBuiltinLenient(cfg StringToTypeConfig, builtinBaseTypeID string) (st *Statement) {

	cfg.AddImport(pkgStrconv, "strconv")
	targetType := cfg.FieldType(cfg.Arg.TypeID, 0, false)
	needsCast := namedBuiltinNeedsCast(cfg.Project, cfg.Arg.TypeID, builtinBaseTypeID)
	switch builtinBaseTypeID {
	case "string":
		return assignNamedBuiltinParsedValueInline(cfg, targetType, cfg.From, needsCast)
	case "bool":
		return buildNamedBuiltinParseIf(cfg, targetType, needsCast, "_parsed", Qual(pkgStrconv, "ParseBool").Call(cfg.From))
	case "int":
		return buildNamedBuiltinParseIf(cfg, targetType, needsCast, "_parsed", Qual(pkgStrconv, "Atoi").Call(cfg.From))
	case "int64":
		return buildNamedBuiltinParseIf(cfg, targetType, needsCast, "_parsed", Qual(pkgStrconv, "ParseInt").Call(cfg.From, Lit(10), Lit(64)))
	case "int32":
		return buildNamedBuiltinParseNarrowIf(cfg, targetType, needsCast, "_parsed64", "_parsed", Qual(pkgStrconv, "ParseInt").Call(cfg.From, Lit(10), Lit(32)), Int32())
	case "uint":
		return buildNamedBuiltinParseNarrowIf(cfg, targetType, needsCast, "_parsed64", "_parsed", Qual(pkgStrconv, "ParseUint").Call(cfg.From, Lit(10), Lit(64)), Uint())
	case "uint64":
		return buildNamedBuiltinParseIf(cfg, targetType, needsCast, "_parsed", Qual(pkgStrconv, "ParseUint").Call(cfg.From, Lit(10), Lit(64)))
	case "uint32":
		return buildNamedBuiltinParseNarrowIf(cfg, targetType, needsCast, "_parsed64", "_parsed", Qual(pkgStrconv, "ParseUint").Call(cfg.From, Lit(10), Lit(32)), Uint32())
	case "float64":
		return buildNamedBuiltinParseIf(cfg, targetType, needsCast, "_parsed", Qual(pkgStrconv, "ParseFloat").Call(cfg.From, Lit(64)))
	case "float32":
		return buildNamedBuiltinParseNarrowIf(cfg, targetType, needsCast, "_parsed64", "_parsed", Qual(pkgStrconv, "ParseFloat").Call(cfg.From, Lit(32)), Float32())
	default:
		return buildStringToTypeJSON(cfg)
	}
}

func buildNamedBuiltinParseIf(cfg StringToTypeConfig, targetType *Statement, needsCast bool, parsedVarName string, parseCall *Statement) (st *Statement) {

	body := []Code{
		assignNamedBuiltinParsedValueInline(cfg, targetType, Id(parsedVarName), needsCast),
	}
	return If(List(Id(parsedVarName), Id("_err")).Op(":=").Add(parseCall).Op(";").Id("_err").Op("==").Nil()).Block(body...)
}

func buildNamedBuiltinParseNarrowIf(cfg StringToTypeConfig, targetType *Statement, needsCast bool, parsedWideVarName string, parsedNarrowVarName string, parseCall *Statement, narrowType *Statement) (st *Statement) {

	body := []Code{
		Id(parsedNarrowVarName).Op(":=").Add(narrowType).Call(Id(parsedWideVarName)),
		assignNamedBuiltinParsedValueInline(cfg, targetType, Id(parsedNarrowVarName), needsCast),
	}
	return If(List(Id(parsedWideVarName), Id("_err")).Op(":=").Add(parseCall).Op(";").Id("_err").Op("==").Nil()).Block(body...)
}

func namedBuiltinNeedsCast(project *model.Project, typeID string, builtinBaseTypeID string) (needsCast bool) {

	needsCast = true
	if project == nil || builtinBaseTypeID == "" {
		return
	}
	var typ *model.Type
	var ok bool
	if typ, ok = project.Types[typeID]; !ok || typ == nil {
		return
	}
	if typ.UnderlyingTypeID == builtinBaseTypeID {
		needsCast = false
		return
	}
	return
}

func assignNamedBuiltinParsedValueInline(cfg StringToTypeConfig, targetType *Statement, parsedValue Code, needsCast bool) (st *Statement) {

	valueExpr := parsedValue
	if needsCast {
		valueExpr = Add(targetType).Call(parsedValue)
	}
	if cfg.Arg.NumberOfPointers > 0 {
		return Block(
			Id("_value").Op(":=").Add(valueExpr),
			Add(cfg.Id).Op("=").Op("&").Id("_value"),
		)
	}
	return Add(cfg.Id).Op("=").Add(valueExpr)
}

func IsBuiltinTypeID(typeID string) (ok bool) {

	switch typeID {
	case "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128",
		"bool", "byte", "rune", "error", "any":
		return true
	}
	return false
}
