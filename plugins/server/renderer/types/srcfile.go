// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package types

type SrcFile interface {
	ImportName(pkgPath, name string)
	ImportAlias(pkgPath, alias string)
}
