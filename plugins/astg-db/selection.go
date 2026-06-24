package main

import (
	"fmt"

	"tgp/core"
	"tgp/core/i18n"
	"tgp/internal/cdb"
	"tgp/internal/model"
)

func selectRef(idx *cdb.Index) (refStr string, err error) {

	refs := cdb.ListRefs(idx)
	if len(refs) == 0 {
		err = fmt.Errorf("%s", i18n.Msg("contracts db empty"))
		return
	}

	var selected []string
	selected, err = core.InteractiveSelect(i18n.Msg("Select project (project@version)"), refs, false, nil)
	if err != nil {
		return
	}
	if len(selected) == 0 {
		err = fmt.Errorf("%s", i18n.Msg("no project selected"))
		return
	}

	refStr = selected[0]
	return
}

func selectContracts(project *model.Project) (contractNames []string, err error) {

	if len(project.Contracts) == 0 {
		return
	}

	options := make([]string, 0, len(project.Contracts))
	for _, contract := range project.Contracts {
		options = append(options, contract.Name)
	}

	var selected []string
	selected, err = core.InteractiveSelect(i18n.Msg("Select contracts"), options, true, options)
	if err != nil {
		return
	}
	if len(selected) == 0 {
		err = fmt.Errorf("%s", i18n.Msg("no contracts selected"))
		return
	}

	contractNames = selected
	return
}
