// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *Renderer) renderHandlers() (err error) {

	file := NewSrcFile(r.pkgName)
	for _, contract := range r.contracts() {
		for _, method := range contract.Methods {
			message, _ := model.MethodKafkaMessageArg(r.project, contract, method)
			element, _ := model.TypeRefKafkaMessageElement(&message.TypeRef)
			if typ := r.project.Types[element.TypeID]; typ != nil && typ.ImportPkgPath != "" {
				file.ImportName(typ.ImportPkgPath, filepath.Base(typ.ImportPkgPath))
			}
		}
	}
	for _, contract := range r.contracts() {
		for _, suffix := range []string{"", "Meta", "Slice", "Batch"} {
			handlerName := contract.Name + suffix + "Handler"
			file.Line().Commentf("%s описывает обработчик событий контракта %s.", handlerName, contract.Name)
			file.Type().Id(handlerName).InterfaceFunc(func(group *Group) {

				for _, method := range contract.Methods {
					eventType := r.eventType(contract, method)
					switch suffix {
					case "Meta":
						group.Id(method.Name).Params(
							Id("ctx").Qual("context", "Context"),
							Id("event").Add(eventType),
							Id("meta").Id("Meta"),
						).Params(Id("err").Error())
					case "Slice":
						group.Id(method.Name).Params(
							Id("ctx").Qual("context", "Context"),
							Id("events").Index().Add(eventType),
						).Params(Id("err").Error())
					case "Batch":
						group.Id(method.Name).Params(
							Id("ctx").Qual("context", "Context"),
							Id("batch").Id("Batch").Types(eventType),
						).Params(Id("err").Error())
					default:
						group.Id(method.Name).Params(
							Id("ctx").Qual("context", "Context"),
							Id("event").Add(eventType),
						).Params(Id("err").Error())
					}
				}
			})
			optionName := contract.Name + suffix
			file.Line().Commentf("%s регистрирует %s-форму обработчика контракта %s.", optionName, handlerFormName(suffix), contract.Name)
			file.Func().Id(optionName).Params(Id("handler").Id(handlerName)).Id("Option").Block(
				Return(Func().Params(Id("setup").Op("*").Id("setup")).Block(
					If(Id("handler").Op("==").Nil()).Block(
						Id("setup").Dot("err").Op("=").
							Qual("fmt", "Errorf").Call(Lit("kafka subscriber: "+contract.Name+" handler is nil")),
						Return(),
					),
					If(List(Id("_"), Id("exists")).Op(":=").Id("setup").Dot("handlers").Index(Lit(contract.Name)), Id("exists")).Block(
						Id("setup").Dot("handlerConflict").Op("=").Lit(contract.Name),
						Return(),
					),
					Id("setup").Dot("handlers").Index(Lit(contract.Name)).Op("=").
						Id("registeredHandler").Values(Dict{
						Id("kind"):    Lit(suffix),
						Id("handler"): Id("handler"),
					}),
				)),
			)
		}
	}
	return file.Save(filepath.Join(r.outDir, "handlers.go"))
}

func handlerFormName(suffix string) (name string) {

	if suffix == "" {
		return "обычную"
	}
	return strings.ToLower(suffix)
}
