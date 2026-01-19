// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package helper

import (
	"fmt"

	"tgp/core/data"
	"tgp/core/i18n"
)

// CreateResponse создает новый response storage и устанавливает output.
func CreateResponse(output string) (response data.Storage, err error) {

	response = data.NewStorage()
	if err = response.Set("out", output); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.Msg("failed to set response"), err)
	}

	return
}
