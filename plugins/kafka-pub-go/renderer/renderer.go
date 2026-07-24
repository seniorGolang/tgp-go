// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tgp/internal/generated"
	kafkaruntime "tgp/internal/kafka"
	"tgp/internal/model"
)

// Renderer формирует исходники пакета Kafka-издателя.
type Renderer struct {
	project          *model.Project
	outDir           string
	targetModulePath string
	outputRelPath    string
	contracts        []*model.Contract
}

// New создаёт рендерер только для Kafka-контрактов.
func New(project *model.Project, outDir string, targetModulePath string, outputRelPath string) (renderer *Renderer) {

	renderer = &Renderer{project: project, outDir: outDir, targetModulePath: targetModulePath, outputRelPath: outputRelPath}
	for _, contract := range model.ContractsSorted(project.Contracts) {
		if model.ContractIsKafka(project, contract) {
			renderer.contracts = append(renderer.contracts, contract)
		}
	}
	return renderer
}

// Render создаёт runtime и адаптеры контрактов.
func (r *Renderer) Render() (err error) {

	if err = os.MkdirAll(r.outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err = kafkaruntime.WriteCodec(r.outDir, filepath.Base(r.outDir), r.codecOptions()); err != nil {
		return err
	}
	if err = kafkaruntime.WriteProduce(r.outDir, filepath.Base(r.outDir)); err != nil {
		return err
	}
	if err = kafkaruntime.WriteSecurity(r.outDir, filepath.Base(r.outDir)); err != nil {
		return err
	}
	if err = r.renderVersion(); err != nil {
		return err
	}
	if err = r.renderClient(); err != nil {
		return err
	}
	if err = r.renderOptions(); err != nil {
		return err
	}
	if err = r.renderAdapters(); err != nil {
		return err
	}
	if r.hasAnnotation(model.TagMetrics) {
		if err = r.write("metrics.go", r.metricsSource()); err != nil {
			return err
		}
	}
	if r.hasAnnotation(model.TagTrace) {
		if err = r.write("tracing.go", r.tracingSource()); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) codecOptions() (options kafkaruntime.WriteCodecOptions) {

	options.PackageJSON = r.project.Annotations.Value(model.TagPackageJSON, "")
	for _, contract := range r.contracts {
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
	return options
}

func (r *Renderer) hasAnnotation(tag string) (found bool) {

	for _, contract := range r.contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, tag) {
			return true
		}
	}
	return false
}

func (r *Renderer) write(name string, source string) (err error) {

	source = generated.ByToolGatewayComment + source
	var formatted []byte
	if formatted, err = format.Source([]byte(source)); err != nil {
		return fmt.Errorf("format %s: %w", name, err)
	}
	return os.WriteFile(filepath.Join(r.outDir, name), formatted, 0o644)
}

func (r *Renderer) acks() (acks []string) {

	seen := make(map[string]struct{})
	for _, contract := range r.contracts {
		for _, method := range contract.Methods {
			ack := model.MethodKafkaAcks(r.project, contract, method)
			if _, ok := seen[ack]; !ok {
				seen[ack] = struct{}{}
				acks = append(acks, ack)
			}
		}
	}
	sort.Strings(acks)
	return acks
}

func ackField(acks string) (field string) {

	switch acks {
	case model.KafkaAcksNoAck:
		return "noAck"
	case model.KafkaAcksLeader:
		return "leaderAck"
	default:
		return "allISRAcks"
	}
}

func lowerFirst(value string) (result string) {

	if value == "" {
		return value
	}
	return strings.ToLower(value[:1]) + value[1:]
}
