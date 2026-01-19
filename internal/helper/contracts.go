// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package helper

import (
	"fmt"
	"strings"

	"tgp/internal/model"
)

// FilterContracts фильтрует контракты проекта по списку имен или ID.
// Если список пуст, возвращает все контракты без изменений.
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

// FilterContractsByInterfaces фильтрует контракты проекта по списку интерфейсов с поддержкой include/exclude логики.
// Если список пуст, возвращает все контракты без изменений.
// Поддерживает префикс "!" для исключения интерфейсов (exclude).
// Если указаны и include, и exclude одновременно, возвращает ошибку.
// ifaces - список имен интерфейсов, где имена с префиксом "!" означают исключение.
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
		// Режим include: включаем только указанные интерфейсы
		for _, contract := range project.Contracts {
			if contains(include, contract.Name) {
				filteredContracts = append(filteredContracts, contract)
			}
		}
	case len(exclude) != 0:
		// Режим exclude: исключаем указанные интерфейсы
		for _, contract := range project.Contracts {
			if !contains(exclude, contract.Name) {
				filteredContracts = append(filteredContracts, contract)
			}
		}
	default:
		// Если списки пусты (не должно произойти, но на всякий случай)
		return project.Contracts, nil
	}

	return
}

// contains проверяет, содержится ли строка в слайсе.
func contains(slice []string, item string) (found bool) {

	for _, s := range slice {
		if s == item {
			found = true
			return
		}
	}
	return
}
