// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package main

import (
	"tgp/core"
)

func init() {

	core.InitPlugin(&AstgPlugin{})
	core.SetInitGeneratorInstance(&AstgPlugin{})
}

func main() {

	// Инициализация не требуется для wasip1
}
