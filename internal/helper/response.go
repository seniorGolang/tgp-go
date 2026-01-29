// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package helper

import (
	"fmt"

	"tgp/core/data"
	"tgp/core/i18n"
)

func CreateResponse(output string) (response data.Storage, err error) {

	response = data.NewStorage()
	if err = response.Set("out", output); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.Msg("failed to set response"), err)
	}

	return
}
