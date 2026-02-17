// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package cache

import (
	"tgp/internal/model"
)

type cacheEntry struct {
	Project *model.Project    `json:"project"`
	Files   map[string]string `json:"files"`
}
