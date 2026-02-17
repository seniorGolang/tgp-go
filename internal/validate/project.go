// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package validate

import (
	"fmt"

	"tgp/internal/model"
)

func Project(project *model.Project) (err error) {

	if project == nil {
		return fmt.Errorf("project cannot be nil")
	}
	if project.ModulePath == "" {
		return fmt.Errorf("project.ModulePath cannot be empty")
	}
	return
}

func ContractID(contractID string) (err error) {

	if contractID == "" {
		return fmt.Errorf("contractID cannot be empty")
	}
	return
}

func OutDir(outDir string) (err error) {

	if outDir == "" {
		return fmt.Errorf("outDir cannot be empty")
	}
	return
}

func FindContract(project *model.Project, contractID string) (contract *model.Contract, err error) {

	if project == nil {
		return nil, fmt.Errorf("project cannot be nil")
	}
	if contractID == "" {
		return nil, fmt.Errorf("contractID cannot be empty")
	}

	for _, c := range project.Contracts {
		if c.ID == contractID {
			return c, nil
		}
	}

	return nil, fmt.Errorf("contract %q not found", contractID)
}
