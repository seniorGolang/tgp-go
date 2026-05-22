// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

// clientRPCResultValue — выражение для извлечения результата из exchange-структуры после json.Unmarshal.
func (r *ClientRenderer) clientRPCResultValue(contract *model.Contract, method *model.Method, ret *model.Variable, field exchangeField, responseVar string) (value Code) {

	if model.ResultFieldEmbedded(r.project, contract, method, ret) {
		embedName := model.TypeNameFromTypeID(r.project, ret.TypeID)
		value = Id(responseVar).Dot(embedName)
	} else {
		value = Id(responseVar).Dot(ToCamel(field.name))
	}

	switch {
	case field.numberOfPointers > 0 && ret.NumberOfPointers == 0:
		return Op("*").Add(value)
	case field.numberOfPointers == 0 && ret.NumberOfPointers > 0:
		return Op("&").Add(value)
	default:
		return value
	}
}
