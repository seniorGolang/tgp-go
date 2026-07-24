// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"github.com/dave/jennifer/jen"

	"tgp/internal/model"
)

func (r *Renderer) needsTiming(contract *model.Contract) (needed bool) {

	return model.IsAnnotationSet(r.project, contract, nil, nil, model.TagMetrics) || model.IsAnnotationSet(r.project, contract, nil, nil, model.TagLogger)
}

func (r *Renderer) addStarted(body *[]jen.Code, contract *model.Contract) {

	if r.needsTiming(contract) {
		*body = append(*body, jen.Id("started").Op(":=").Qual("time", "Now").Call())
	}
}

func (r *Renderer) addTrace(body *[]jen.Code, contract *model.Contract, method *model.Method, records jen.Code) {

	if !model.IsAnnotationSet(r.project, contract, nil, nil, model.TagTrace) {
		return
	}
	*body = append(*body,
		jen.Var().Id("finish").Func().Params(jen.Error()),
		jen.List(jen.Id("ctx"), jen.Id("finish")).Op("=").Id("adapter").Dot("client").Dot("startProduceSpan").Call(jen.Id("ctx"), jen.Lit(contract.Name), jen.Lit(method.Name), jen.Lit(model.MethodKafkaTopic(r.project, contract, method)), records),
		jen.Defer().Func().Params().Block(jen.Id("finish").Call(jen.Id("err"))).Call(),
	)
}

func (r *Renderer) addObservation(body *[]jen.Code, contract *model.Contract, method *model.Method, records jen.Code, cause string, outcomes jen.Code) {

	codes := r.observationCodes(contract, method, records, cause, outcomes, false)
	*body = append(*body, codes...)
}

func (r *Renderer) addEncodeFailure(group *jen.Group, contract *model.Contract, method *model.Method, records jen.Code) {

	for _, code := range r.observationCodes(contract, method, records, "encode", nil, true) {
		group.Add(code)
	}
}

func (r *Renderer) observationCodes(contract *model.Contract, method *model.Method, records jen.Code, cause string, outcomes jen.Code, failureOnly bool) (codes []jen.Code) {

	hasMetrics := model.IsAnnotationSet(r.project, contract, nil, nil, model.TagMetrics)
	hasLog := model.IsAnnotationSet(r.project, contract, nil, nil, model.TagLogger)
	topic := model.MethodKafkaTopic(r.project, contract, method)
	if hasMetrics {
		codes = append(codes,
			jen.Id("adapter").Dot("client").Dot("observeProduceCall").Call(
				jen.Lit(contract.Name), jen.Lit(method.Name), jen.Lit(topic),
				records, jen.Qual("time", "Since").Call(jen.Id("started")), jen.Id("err"), jen.Lit(cause),
			),
		)
		if outcomes != nil {
			codes = append(codes, jen.Id("adapter").Dot("client").Dot("observeProduceRecords").Call(
				jen.Lit(contract.Name), jen.Lit(method.Name), jen.Lit(topic), outcomes,
			))
		}
	}
	if !hasLog {
		return codes
	}
	fields := []jen.Code{
		jen.Lit("tgp.contract"), jen.Lit(contract.Name),
		jen.Lit("tgp.method"), jen.Lit(method.Name),
		jen.Lit("messaging.destination"), jen.Lit(topic),
		jen.Lit("tgp.records"), records,
		jen.Lit("tgp.duration"), jen.Qual("time", "Since").Call(jen.Id("started")),
	}
	errorFields := append(append([]jen.Code{}, fields...), jen.Lit("error"), jen.Id("err"))
	errorLog := jen.Id("adapter").Dot("client").Dot("log").Dot("Error").Call(append([]jen.Code{jen.Lit("kafka produce failed")}, errorFields...)...)
	if failureOnly {
		codes = append(codes, errorLog)
		return codes
	}
	codes = append(codes, jen.If(jen.Id("err").Op("!=").Nil()).Block(errorLog).Else().Block(
		jen.Id("adapter").Dot("client").Dot("log").Dot("Info").Call(append([]jen.Code{jen.Lit("kafka produce completed")}, fields...)...),
	))
	return codes
}
