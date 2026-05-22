// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"context"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/common"
	"tgp/internal/converter"
	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *ClientRenderer) httpResponseMergeHeadersAndCookies(bg *Group, ctx context.Context, contract *model.Contract, method *model.Method, assignToReturnVars bool) {

	headerMap := model.HTTPResultHeaderMapForResponse(r.project, contract, method)
	cookieMap := model.HTTPResultCookieMapForResponse(r.project, contract, method)
	if len(headerMap) == 0 && len(cookieMap) == 0 {
		return
	}

	var ok bool
	var srcFile GoFile
	if srcFile, ok = ctx.Value(keyCode).(GoFile); !ok {
		return
	}
	typeGen := types.NewGenerator(r.project, srcFile)
	jsonPkg := r.getPackageJSON(contract)

	valVarPrefix := "_parse"
	target := func(fieldName string) *Statement {
		if assignToReturnVars {
			return Id(ToLowerCamel(fieldName))
		}
		return Id("_response_").Dot(ToCamel(fieldName))
	}
	for retName, headerName := range common.SortedPairs(headerMap) {
		ret := r.resultByName(method, retName)
		if ret == nil {
			continue
		}
		fieldName := ret.Name
		cfg := converter.StringToTypeConfig{
			Project:        r.project,
			From:           Id("httpResp").Dot("Header").Dot("Get").Call(Lit(headerName)),
			Arg:            ret,
			Id:             target(fieldName),
			OptionalAssign: true,
			FieldType:      typeGen.FieldType,
			AddImport:      srcFile.ImportName,
			JSONPkg:        jsonPkg,
			AddTo:          bg,
			ErrVar:         Err(),
			ValVarName:     valVarPrefix + ToCamel(fieldName) + "_",
		}
		if st := converter.BuildStringToType(cfg); st != nil {
			bg.Add(st)
		}
	}
	for retName, cookieName := range common.SortedPairs(cookieMap) {
		ret := r.resultByName(method, retName)
		if ret == nil {
			continue
		}
		fieldName := ret.Name
		cookieVar := "_cookie" + ToCamel(fieldName) + "_"
		bg.Var().Id(cookieVar).String()
		bg.For(List(Id("_"), Id("_c")).Op(":=").Range().Id("httpResp").Dot("Cookies").Call()).
			Block(
				If(Id("_c").Dot("Name").Op("==").Lit(cookieName)).Block(
					Id(cookieVar).Op("=").Id("_c").Dot("Value"),
					Break(),
				),
			)
		cfg := converter.StringToTypeConfig{
			Project:        r.project,
			From:           Id(cookieVar),
			Arg:            ret,
			Id:             target(fieldName),
			OptionalAssign: true,
			FieldType:      typeGen.FieldType,
			AddImport:      srcFile.ImportName,
			JSONPkg:        jsonPkg,
			AddTo:          bg,
			ErrVar:         Err(),
			ValVarName:     valVarPrefix + ToCamel(fieldName) + "_",
		}
		if st := converter.BuildStringToType(cfg); st != nil {
			bg.Add(st)
		}
	}
}
