// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package helper

import (
	"errors"
	"fmt"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/internal/model"
)

func GetProject(request data.Storage) (project *model.Project, err error) {

	if project, err = data.Get[*model.Project](request, "project"); err != nil || project == nil {
		if errors.Is(err, data.ErrNotFound) {
			return nil, errors.New(i18n.Msg("project is required in request"))
		}
		return nil, fmt.Errorf(i18n.Msg("failed to get project")+": %w", err)
	}

	return
}
