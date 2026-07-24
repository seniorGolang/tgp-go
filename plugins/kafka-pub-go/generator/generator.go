// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"log/slog"

	"tgp/internal/model"
	"tgp/internal/validate"
	"tgp/plugins/kafka-pub-go/renderer"
)

// Generate проверяет модель и генерирует Kafka-издатель.
func Generate(project *model.Project, outDir string, targetModulePath string, outputRelPath string) (err error) {

	if err = validate.Project(project); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}
	if err = validate.KafkaProject(project); err != nil {
		return fmt.Errorf("invalid kafka project: %w", err)
	}
	for _, contract := range project.Contracts {
		if err = validate.Contract(contract, project); err != nil {
			return fmt.Errorf("validate contract %q: %w", contract.Name, err)
		}
	}
	if !hasKafkaContracts(project) {
		return fmt.Errorf("no kafka contracts")
	}
	for _, contract := range project.Contracts {
		if !model.ContractIsKafka(project, contract) {
			continue
		}
		for _, method := range contract.Methods {
			if extra := model.MethodKafkaExtraArgs(project, contract, method); len(extra) != 0 {
				slog.Warn("kafka method has arguments outside message/key/headers", slog.String("contract", contract.Name), slog.String("method", method.Name))
			}
		}
	}
	r := renderer.New(project, outDir, targetModulePath, outputRelPath)
	if err = r.Render(); err != nil {
		return fmt.Errorf("render kafka publisher: %w", err)
	}
	return nil
}

func hasKafkaContracts(project *model.Project) (ok bool) {

	for _, contract := range project.Contracts {
		if model.ContractIsKafka(project, contract) {
			return true
		}
	}
	return false
}
