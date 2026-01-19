// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"

	"tgp/internal/model"
)

// AnalysisStats содержит статистику анализа проекта.
type AnalysisStats struct {
	ServicesCount        int
	ImplementationsCount int
	TypesCount           int
}

// analyzeProject выполняет полный анализ проекта после сбора базовых данных.
func analyzeProject(project *model.Project, loader *AutonomousPackageLoader) (stats AnalysisStats, err error) {

	// 1. Поиск сервисов (main файлов)
	if err = findServices(project, loader); err != nil {
		err = fmt.Errorf("failed to find services: %w", err)
		return
	}
	stats.ServicesCount = len(project.Services)

	// 2. Поиск имплементаций контрактов
	if err = findImplementations(project, loader); err != nil {
		err = fmt.Errorf("failed to find implementations: %w", err)
		return
	}
	totalImplementations := 0
	for _, contract := range project.Contracts {
		totalImplementations += len(contract.Implementations)
	}
	stats.ImplementationsCount = totalImplementations

	// 3. Анализ ошибок методов
	if err = analyzeMethodErrors(project, loader); err != nil {
		err = fmt.Errorf("failed to analyze method errors: %w", err)
		return
	}

	// 4. Рекурсивное разбирание всех типов из контрактов
	if err = expandTypesRecursively(project, loader); err != nil {
		err = fmt.Errorf("failed to expand types recursively: %w", err)
		return
	}
	stats.TypesCount = len(project.Types)

	return
}
