// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/kafka"
	"tgp/internal/model"
)

// Renderer генерирует подписчик Kafka из контрактов проекта.
type Renderer struct {
	project *model.Project
	outDir  string
	pkgName string
}

// NewRenderer создаёт генератор исходных файлов.
func NewRenderer(project *model.Project, outDir string, targetModulePath string, outputRelPath string) (renderer *Renderer) {

	return &Renderer{project: project, outDir: outDir, pkgName: filepath.Base(outDir)}
}

// HasKafka сообщает, есть ли в запуске Kafka-контракты.
func (r *Renderer) HasKafka() (ok bool) {

	return len(r.contracts()) != 0
}

func (r *Renderer) contracts() (contracts []*model.Contract) {

	for _, contract := range r.project.Contracts {
		if model.ContractIsKafka(r.project, contract) {
			contracts = append(contracts, contract)
		}
	}
	return model.ContractsSorted(contracts)
}

// RenderAll записывает полный набор файлов подписчика.
func (r *Renderer) RenderAll() (err error) {

	if !r.HasKafka() {
		return fmt.Errorf("kafka contracts are required")
	}
	if err = r.renderRuntime(); err != nil {
		return
	}
	if err = r.renderVersion(); err != nil {
		return
	}
	if err = r.renderOptions(); err != nil {
		return
	}
	if err = r.renderHandlers(); err != nil {
		return
	}
	if err = r.renderSubscriber(); err != nil {
		return
	}
	if r.hasMetrics() {
		if err = writeSource(r.outDir, "metrics.go", r.metricsSource()); err != nil {
			return
		}
	}
	if r.hasTrace() {
		if err = writeSource(r.outDir, "tracing.go", r.tracingSource()); err != nil {
			return
		}
	}
	return nil
}

func (r *Renderer) renderRuntime() (err error) {

	options := kafka.WriteCodecOptions{
		PackageJSON: r.project.Annotations.Value(model.TagPackageJSON, ""),
	}
	for _, contract := range r.contracts() {
		for _, method := range contract.Methods {
			switch model.MethodKafkaCodec(r.project, contract, method) {
			case model.KafkaCodecMsgpack:
				options.IncludeMsgpack = true
			case model.KafkaCodecCBOR:
				options.IncludeCBOR = true
			case model.KafkaCodecYAML:
				options.IncludeYAML = true
			case model.KafkaCodecXML:
				options.IncludeXML = true
			}
		}
	}
	if err = kafka.WriteCodec(r.outDir, r.pkgName, options); err != nil {
		return
	}
	if err = kafka.WriteRecord(r.outDir, r.pkgName); err != nil {
		return
	}
	if err = kafka.WriteSecurity(r.outDir, r.pkgName); err != nil {
		return
	}
	return kafka.WritePoll(r.outDir, r.pkgName)
}

func (r *Renderer) topics() (topics []string) {

	for _, contract := range r.contracts() {
		for _, method := range contract.Methods {
			topics = append(topics, model.MethodKafkaTopic(r.project, contract, method))
		}
	}
	sort.Strings(topics)
	return topics
}

func (r *Renderer) hasMetrics() (ok bool) {

	for _, contract := range r.contracts() {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagMetrics) {
			return true
		}
	}
	return false
}

func (r *Renderer) hasTrace() (ok bool) {

	for _, contract := range r.contracts() {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagTrace) {
			return true
		}
	}
	return false
}

func (r *Renderer) hasLog() (ok bool) {

	for _, contract := range r.contracts() {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagLogger) {
			return true
		}
	}
	return false
}

func (r *Renderer) eventType(contract *model.Contract, method *model.Method) (statement *Statement) {

	message, _ := model.MethodKafkaMessageArg(r.project, contract, method)
	element, _ := model.TypeRefKafkaMessageElement(&message.TypeRef)
	if model.TypeRefIsByteSlice(&element) {
		return Index().Byte()
	}

	name := element.TypeID
	if typ := r.project.Types[element.TypeID]; typ != nil && typ.ImportPkgPath != "" {
		name = typ.TypeName
		statement = Qual(typ.ImportPkgPath, name)
	} else {
		packagePath, typeName, found := strings.Cut(name, ":")
		if found {
			statement = Qual(packagePath, typeName)
		} else {
			statement = Id(name)
		}
	}
	if element.IsSlice {
		return Index().Add(statement)
	}
	return statement
}
