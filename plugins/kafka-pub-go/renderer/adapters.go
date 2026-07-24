// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path/filepath"
	"strings"

	"github.com/dave/jennifer/jen"

	"tgp/internal/model"
)

func (r *Renderer) renderAdapters() (err error) {

	source := newSrcFile(filepath.Base(r.outDir))
	for _, contract := range r.contracts {
		adapter := lowerFirst(contract.Name) + "Adapter"
		source.Type().Id(adapter).Struct(jen.Id("client").Op("*").Id("Client"))
		source.Line()
		source.Func().Params(jen.Id("client").Op("*").Id("Client")).Id(contract.Name).Params().Params(jen.Id("service").Qual(contract.PkgPath, contract.Name)).Block(
			jen.Return(jen.Op("&").Id(adapter).Values(jen.Dict{jen.Id("client"): jen.Id("client")})),
		)
		source.Line()
		for _, method := range contract.Methods {
			r.addAdapterMethod(source, adapter, contract, method)
			source.Line()
		}
	}
	return source.Save(filepath.Join(r.outDir, "adapters.go"))
}

func (r *Renderer) addAdapterMethod(source *GoFile, adapter string, contract *model.Contract, method *model.Method) {

	message, _ := model.MethodKafkaMessageArg(r.project, contract, method)
	isBatch := (message.IsSlice || message.IsEllipsis) && !model.TypeRefIsByteSlice(&message.TypeRef)
	parameters := make([]jen.Code, 0, len(method.Args))
	for _, arg := range method.Args {
		parameters = append(parameters, jen.Id(arg.Name).Add(r.typeCode(&arg.TypeRef, true)))
	}
	body := []jen.Code{
		jen.If(jen.Id("err").Op("=").Id("ctx").Dot("Err").Call(), jen.Id("err").Op("!=").Nil()).Block(jen.Return()),
	}
	if isBatch {
		body = append(body, r.batchAdapterBody(contract, method, message.Name)...)
	} else {
		body = append(body, r.singleAdapterBody(contract, method, message.Name)...)
	}
	source.Func().Params(jen.Id("adapter").Op("*").Id(adapter)).Id(method.Name).Params(parameters...).Params(jen.Id("err").Error()).Block(body...)
}

func (r *Renderer) batchAdapterBody(contract *model.Contract, method *model.Method, messageName string) (body []jen.Code) {

	records := jen.Len(jen.Id("records"))
	body = append(body, jen.If(jen.Len(jen.Id(messageName)).Op("==").Lit(0)).Block(jen.Return(jen.Nil())))
	r.addStarted(&body, contract)
	body = append(body,
		jen.Id("records").Op(":=").Make(jen.Index().Op("*").Qual(kgoPath, "Record"), jen.Lit(0), jen.Len(jen.Id(messageName))),
		jen.Id("codec").Op(":=").Id("adapter").Dot("client").Dot("codecs").Index(jen.Lit(model.MethodKafkaCodec(r.project, contract, method))),
		jen.For(jen.List(jen.Id("index"), jen.Id("message")).Op(":=").Range().Id(messageName)).BlockFunc(func(group *jen.Group) {
			if !r.needsBatchIndex(contract, method) {
				group.Id("_").Op("=").Id("index")
			}
			r.addRecordBuild(group, contract, method, jen.Id("message"), jen.Id("index"), jen.Len(jen.Id(messageName)))
			group.Id("records").Op("=").Append(jen.Id("records"), jen.Id("record"))
		}),
		jen.Var().Id("client").Op("*").Qual(kgoPath, "Client"),
		jen.If(jen.List(jen.Id("client"), jen.Id("err")).Op("=").Id("adapter").Dot("client").Dot("kafkaClient").Call(jen.Lit(model.MethodKafkaAcks(r.project, contract, method))), jen.Id("err").Op("!=").Nil()).Block(jen.Return()),
	)
	r.addTrace(&body, contract, method, records)
	body = append(body,
		jen.Var().Id("outcomes").Index().Id("produceOutcome"),
		jen.List(jen.Id("outcomes"), jen.Id("err")).Op("=").Id("produceAllAndWait").Call(jen.Id("ctx"), jen.Id("client"), jen.Id("records"), jen.Id("adapter").Dot("client").Dot("produceHooks").Call()),
	)
	r.addObservation(&body, contract, method, records, "produce", jen.Id("outcomes"))
	body = append(body, jen.Return())
	return body
}

func (r *Renderer) singleAdapterBody(contract *model.Contract, method *model.Method, messageName string) (body []jen.Code) {

	r.addStarted(&body, contract)
	body = append(body, jen.Id("codec").Op(":=").Id("adapter").Dot("client").Dot("codecs").Index(jen.Lit(model.MethodKafkaCodec(r.project, contract, method))))
	body = append(body, r.recordBuildCodes(contract, method, jen.Id(messageName), jen.Lit(0), jen.Lit(1))...)
	body = append(body,
		jen.Var().Id("client").Op("*").Qual(kgoPath, "Client"),
		jen.If(jen.List(jen.Id("client"), jen.Id("err")).Op("=").Id("adapter").Dot("client").Dot("kafkaClient").Call(jen.Lit(model.MethodKafkaAcks(r.project, contract, method))), jen.Id("err").Op("!=").Nil()).Block(jen.Return()),
	)
	r.addTrace(&body, contract, method, jen.Lit(1))
	body = append(body,
		jen.Id("outcome").Op(":=").Id("produceAndWait").Call(jen.Id("ctx"), jen.Id("client"), jen.Id("record"), jen.Id("adapter").Dot("client").Dot("produceHooks").Call()),
		jen.Id("err").Op("=").Id("outcome").Dot("err"),
	)
	r.addObservation(&body, contract, method, jen.Lit(1), "produce", jen.Index().Id("produceOutcome").Values(jen.Id("outcome")))
	body = append(body, jen.Return())
	return body
}

func (r *Renderer) recordBuildCodes(contract *model.Contract, method *model.Method, value jen.Code, index jen.Code, records jen.Code) (codes []jen.Code) {

	codes = append(codes,
		jen.Var().Id("data").Index().Byte(),
		jen.If(jen.List(jen.Id("data"), jen.Id("err")).Op("=").Id("codec").Dot("Marshal").Call(value), jen.Id("err").Op("!=").Nil()).BlockFunc(func(block *jen.Group) {
			r.addEncodeFailure(block, contract, method, records)
			block.Return()
		}),
		jen.Var().Id("key").Index().Byte(),
	)
	keyName := model.MethodKafkaKeyArg(r.project, contract, method)
	if keyName != "" {
		codes = append(codes, jen.If(jen.List(jen.Id("key"), jen.Id("err")).Op("=").Id("keyBytes").Call(r.indexedCode(method, keyName, index)), jen.Id("err").Op("!=").Nil()).BlockFunc(func(block *jen.Group) {
			r.addEncodeFailure(block, contract, method, records)
			block.Return()
		}))
	}
	headers := model.MethodKafkaHeaderItems(r.project, contract, method)
	if len(headers) == 0 {
		codes = append(codes, jen.Var().Id("headers").Index().Qual(kgoPath, "RecordHeader"))
	} else {
		codes = append(codes, jen.Id("headers").Op(":=").Make(jen.Index().Qual(kgoPath, "RecordHeader"), jen.Lit(0), jen.Lit(len(headers))))
		codes = append(codes, jen.Var().Id("header").Index().Byte())
		for _, item := range headers {
			item := item
			codes = append(codes,
				jen.If(jen.List(jen.Id("header"), jen.Id("err")).Op("=").Id("headerBytes").Call(r.indexedCode(method, item.Arg, index)), jen.Id("err").Op("!=").Nil()).BlockFunc(func(block *jen.Group) {
					r.addEncodeFailure(block, contract, method, records)
					block.Return()
				}),
				jen.Id("headers").Op("=").Append(jen.Id("headers"), jen.Qual(kgoPath, "RecordHeader").Values(jen.Dict{jen.Id("Key"): jen.Lit(item.Key), jen.Id("Value"): jen.Id("header")})),
			)
		}
	}
	codes = append(codes, jen.Id("record").Op(":=").Op("&").Qual(kgoPath, "Record").Values(jen.Dict{
		jen.Id("Topic"):   jen.Lit(model.MethodKafkaTopic(r.project, contract, method)),
		jen.Id("Key"):     jen.Id("key"),
		jen.Id("Headers"): jen.Id("headers"),
		jen.Id("Value"):   jen.Id("data"),
	}))
	return codes
}

func (r *Renderer) addRecordBuild(group *jen.Group, contract *model.Contract, method *model.Method, value jen.Code, index jen.Code, records jen.Code) {

	for _, code := range r.recordBuildCodes(contract, method, value, index, records) {
		group.Add(code)
	}
}

func (r *Renderer) needsBatchIndex(contract *model.Contract, method *model.Method) (needed bool) {

	keyName := model.MethodKafkaKeyArg(r.project, contract, method)
	if keyName != "" && r.isIndexedArg(method, keyName) {
		return true
	}
	for _, item := range model.MethodKafkaHeaderItems(r.project, contract, method) {
		if r.isIndexedArg(method, item.Arg) {
			return true
		}
	}
	return false
}

func (r *Renderer) isIndexedArg(method *model.Method, name string) (ok bool) {

	for _, arg := range method.Args {
		if arg.Name == name && ((arg.IsSlice && arg.TypeID == "string") || model.TypeRefIsByteSliceSlice(&arg.TypeRef)) {
			return true
		}
	}
	return false
}

func (r *Renderer) indexedCode(method *model.Method, name string, index jen.Code) (value jen.Code) {

	if r.isIndexedArg(method, name) {
		return jen.Id(name).Index(index)
	}
	return jen.Id(name)
}

func (r *Renderer) typeCode(reference *model.TypeRef, ellipsis bool) (result jen.Code) {

	prefix := new(jen.Statement)
	for pointer := 0; pointer < reference.NumberOfPointers; pointer++ {
		prefix.Op("*")
	}
	if reference.IsEllipsis && ellipsis {
		prefix.Op("...")
	} else if reference.IsSlice {
		prefix.Index()
	} else if reference.ArrayLen > 0 {
		prefix.Index(jen.Lit(reference.ArrayLen))
	}
	if reference.MapKey != nil && reference.MapValue != nil {
		return prefix.Map(r.typeCode(reference.MapKey, false)).Add(r.typeCode(reference.MapValue, false))
	}
	if reference.TypeID == "context:Context" || reference.TypeID == "context.Context" {
		return prefix.Qual("context", "Context")
	}
	if reference.TypeID == "[]byte" {
		return prefix.Index().Byte()
	}
	if typ := r.project.Types[reference.TypeID]; typ != nil && typ.ImportPkgPath != "" {
		return prefix.Qual(typ.ImportPkgPath, typ.TypeName)
	}
	if path, name, found := strings.Cut(reference.TypeID, ":"); found {
		return prefix.Qual(path, name)
	}
	return prefix.Id(reference.TypeID)
}
