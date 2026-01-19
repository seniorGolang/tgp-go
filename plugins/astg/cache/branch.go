// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package cache

import (
	"tgp/plugins/astg/marker"
)

// NormalizeBranch нормализует имя ветки (обертка над marker.NormalizeBranchName).
func NormalizeBranch(branch string) string {

	return marker.NormalizeBranchName(branch)
}
