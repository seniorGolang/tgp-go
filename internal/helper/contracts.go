// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package helper

import (
	"fmt"
	"strings"

	"tgp/internal/model"
)

func FilterContracts(project *model.Project, filterNames []string) (filteredContracts []*model.Contract) {

	if len(filterNames) == 0 {
		return project.Contracts
	}

	filteredContracts = make([]*model.Contract, 0)
	for _, contract := range project.Contracts {
		for _, filterName := range filterNames {
			if contract.Name == filterName || contract.ID == filterName {
				filteredContracts = append(filteredContracts, contract)
				break
			}
		}
	}

	return
}

// FilterContractsByInterfaces: префикс "!" в имени интерфейса означает exclude; include и exclude одновременно не допускаются.
func FilterContractsByInterfaces(project *model.Project, ifaces []string) (filteredContracts []*model.Contract, err error) {

	if len(ifaces) == 0 {
		return project.Contracts, nil
	}

	include := make([]string, 0, len(ifaces))
	exclude := make([]string, 0, len(ifaces))
	for _, iface := range ifaces {
		if strings.HasPrefix(iface, "!") {
			exclude = append(exclude, strings.TrimPrefix(iface, "!"))
			continue
		}
		include = append(include, iface)
	}

	if len(include) != 0 && len(exclude) != 0 {
		err = fmt.Errorf("include and exclude cannot be set at same time (%v | %v)", include, exclude)
		return
	}

	filteredContracts = make([]*model.Contract, 0, len(project.Contracts))

	switch {
	case len(include) != 0:
		for _, contract := range project.Contracts {
			if contains(include, contract.Name) {
				filteredContracts = append(filteredContracts, contract)
			}
		}
	case len(exclude) != 0:
		for _, contract := range project.Contracts {
			if !contains(exclude, contract.Name) {
				filteredContracts = append(filteredContracts, contract)
			}
		}
	default:
		return project.Contracts, nil
	}

	return
}

func contains(slice []string, item string) (found bool) {

	for _, s := range slice {
		if s == item {
			found = true
			return
		}
	}
	return
}
