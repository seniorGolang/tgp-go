// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"

	"tgp/internal/model"
)

func analyzeProject(project *model.Project, loader *AutonomousPackageLoader) (err error) {

	if err = findServices(project, loader); err != nil {
		err = fmt.Errorf("failed to find services: %w", err)
		return
	}

	if err = findImplementations(project, loader); err != nil {
		err = fmt.Errorf("failed to find implementations: %w", err)
		return
	}

	if err = analyzeMethodErrors(project, loader); err != nil {
		err = fmt.Errorf("failed to analyze method errors: %w", err)
		return
	}

	if err = expandTypesRecursively(project, loader); err != nil {
		err = fmt.Errorf("failed to expand types recursively: %w", err)
		return
	}

	return
}
