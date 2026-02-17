// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package helper

import (
	"errors"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/internal/common"
)

func GetOutput(request data.Storage) (output string, err error) {

	if output, err = data.Get[string](request, "out"); err != nil || output == "" {
		return "", errors.New(i18n.Msg("out option is required and must be a string"))
	}
	return common.NormalizeWASMPath(output), err
}
