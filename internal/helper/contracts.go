// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package helper

import (
	"fmt"
	"strings"

	"tgp/internal/model"
)

// FilterContractsByInterfaces: префикс "!" в имени означает exclude; include и exclude одновременно не допускаются.
// Имена сопоставляются с Contract.Name и Contract.ID.
func FilterContractsByInterfaces(project *model.Project, ifaces []string) (filteredContracts []*model.Contract, err error) {

	if len(ifaces) == 0 {
		return project.Contracts, nil
	}

	include := make([]string, 0, len(ifaces))
	exclude := make([]string, 0, len(ifaces))
	for _, iface := range ifaces {
		if after, ok := strings.CutPrefix(iface, "!"); ok {
			exclude = append(exclude, after)
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
			if contractMatches(contract, include) {
				filteredContracts = append(filteredContracts, contract)
			}
		}
	case len(exclude) != 0:
		for _, contract := range project.Contracts {
			if !contractMatches(contract, exclude) {
				filteredContracts = append(filteredContracts, contract)
			}
		}
	default:
		return project.Contracts, nil
	}

	return
}

// ProjectWithFilter возвращает shallow-копию project с отфильтрованными контрактами.
// При пустом ifaces возвращает тот же project без копии. Исходный project не мутируется.
func ProjectWithFilter(project *model.Project, ifaces []string) (out *model.Project, err error) {

	if len(ifaces) == 0 {
		return project, nil
	}

	var filtered []*model.Contract
	if filtered, err = FilterContractsByInterfaces(project, ifaces); err != nil {
		return
	}

	clone := *project
	clone.Contracts = filtered
	out = &clone
	return
}

func contractMatches(contract *model.Contract, names []string) (matched bool) {

	for _, name := range names {
		if contract.Name == name || contract.ID == name {
			matched = true
			return
		}
	}
	return
}
