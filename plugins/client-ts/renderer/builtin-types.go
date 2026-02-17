// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"strings"

	"tgp/internal/model"
)

// builtinExact задаёт точный маппинг (последний сегмент пакета:имя_типа) -> встроенный TS-тип.
var builtinExact = map[string]string{
	"time:Time":     "Date",
	"time:Duration": "number",
}

// builtinPkg задаёт маппинг по пакету: любой тип из пакета с таким последним сегментом пути -> TS-тип.
var builtinPkg = map[string]string{
	"uuid": "string",
}

func lastSegment(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}
	i := strings.LastIndex(pkgPath, "/")
	if i >= 0 {
		return pkgPath[i+1:]
	}
	return pkgPath
}

func keyFromTypeID(typeID string) (pkgSeg, typeName string) {
	colon := strings.LastIndex(typeID, ":")
	if colon <= 0 {
		return "", ""
	}
	return lastSegment(typeID[:colon]), typeID[colon+1:]
}

// goBuiltinTSType возвращает встроенный TS-тип для внешнего Go-типа (uuid.UUID, time.Time, time.Duration и т.п.).
// typ может быть nil, если тип не найден в project.Types (тогда используется typeID).
func goBuiltinTSType(typeID string, typ *model.Type) (tsType string, ok bool) {
	var pkgSeg, typeName string
	if typ != nil {
		pkgSeg = lastSegment(typ.ImportPkgPath)
		typeName = typ.TypeName
	} else {
		pkgSeg, typeName = keyFromTypeID(typeID)
	}
	if pkgSeg == "" && typeName == "" {
		return "", false
	}
	key := pkgSeg + ":" + typeName
	if t, has := builtinExact[key]; has {
		return t, true
	}
	if t, has := builtinPkg[pkgSeg]; has {
		return t, true
	}
	return "", false
}
