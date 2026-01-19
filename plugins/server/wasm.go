package main

import (
	"tgp/core"
)

func init() {

	core.InitPlugin(&ServerPlugin{})
}

func main() {

	// Инициализация не требуется для wasip1
}
