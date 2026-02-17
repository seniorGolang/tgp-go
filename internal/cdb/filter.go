package cdb

import (
	"tgp/internal/model"
)

// FilterProject оставляет в проекте только контракты из списка имён и связанные сервисы.
// Если contractNames пустой — контракты не фильтруются.
func FilterProject(project *model.Project, contractNames []string) (out *model.Project) {

	if len(contractNames) == 0 {
		return project
	}

	nameSet := make(map[string]bool)
	for _, n := range contractNames {
		nameSet[n] = true
	}

	var filteredContracts []*model.Contract
	for _, c := range project.Contracts {
		if nameSet[c.Name] || nameSet[c.ID] {
			filteredContracts = append(filteredContracts, c)
		}
	}

	contractIDSet := make(map[string]bool)
	for _, c := range filteredContracts {
		contractIDSet[c.ID] = true
		contractIDSet[c.Name] = true
	}

	var filteredServices []*model.Service
	for _, s := range project.Services {
		keep := false
		for _, id := range s.ContractIDs {
			if contractIDSet[id] {
				keep = true
				break
			}
		}
		if !keep {
			continue
		}
		var ids []string
		for _, id := range s.ContractIDs {
			if contractIDSet[id] {
				ids = append(ids, id)
			}
		}
		filteredServices = append(filteredServices, &model.Service{
			Name:        s.Name,
			MainPath:    s.MainPath,
			ContractIDs: ids,
		})
	}

	return &model.Project{
		Version:      project.Version,
		ModulePath:   project.ModulePath,
		ContractsDir: project.ContractsDir,
		Git:          project.Git,
		Annotations:  project.Annotations,
		Services:     filteredServices,
		Contracts:    filteredContracts,
		Types:        project.Types,
		ExcludeDirs:  project.ExcludeDirs,
		ProjectID:    project.ProjectID,
	}
}
