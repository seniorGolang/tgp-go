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
