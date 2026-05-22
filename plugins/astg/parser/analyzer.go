// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"log/slog"

	"tgp/internal/model"
)

func analyzeProject(project *model.Project, loader *AutonomousPackageLoader) (err error) {

	defer traceRecover("analyzeProject")
	traceBegin("analyzeProject", slog.Int("contracts", len(project.Contracts)))

	if err = findServices(project, loader); err != nil {
		return fmt.Errorf("failed to find services: %w", err)
	}
	traceStep("findServices done", slog.Int("services", len(project.Services)))

	if err = findImplementations(project, loader); err != nil {
		return fmt.Errorf("failed to find implementations: %w", err)
	}
	traceStep("findImplementations done")

	if err = analyzeMethodErrors(project, loader); err != nil {
		return fmt.Errorf("failed to analyze method errors: %w", err)
	}
	traceStep("analyzeMethodErrors done")

	if err = expandTypesRecursively(project, loader); err != nil {
		return fmt.Errorf("failed to expand types recursively: %w", err)
	}
	traceStep("expandTypesRecursively done", slog.Int("types", len(project.Types)))

	traceEnd("analyzeProject")
	return
}
