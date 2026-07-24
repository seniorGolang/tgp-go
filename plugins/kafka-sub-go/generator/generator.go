// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"

	"tgp/internal/model"
	"tgp/internal/validate"
	"tgp/plugins/kafka-sub-go/renderer"
)

// Generate валидирует модель и генерирует Kafka-подписчик.
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
	render := renderer.NewRenderer(project, outDir, targetModulePath, outputRelPath)
	if !render.HasKafka() {
		return fmt.Errorf("kafka-sub-go requires at least one @tg kafka contract")
	}
	if err = render.RenderAll(); err != nil {
		return fmt.Errorf("render kafka subscriber: %w", err)
	}
	return nil
}
