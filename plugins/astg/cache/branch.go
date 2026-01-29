// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package cache

import (
	"tgp/plugins/astg/marker"
)

func NormalizeBranch(branch string) string {

	return marker.NormalizeBranchName(branch)
}
