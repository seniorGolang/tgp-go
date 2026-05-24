// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"strings"

	"tgp/internal/model"
	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) jsonRPCHTTPPrefix() (prefix string) {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			return model.GetAnnotationValue(r.project, contract, nil, nil, model.TagHttpPrefix, "")
		}
	}
	return ""
}

// emitJoinEndpointPrefixTS генерирует resultVar = TrimRight(endpoint, "/") + "/" + http-prefix (как Go-клиент).
func (r *ClientRenderer) emitJoinEndpointPrefixTS(cg *tsg.Group, endpointVar string, prefix string, resultVar string) {

	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		cg.Add(tsg.NewStatement().Const(resultVar).Op("=").Id(endpointVar).Semicolon())

		return
	}

	trimmedBase := tsg.NewStatement().
		Id(endpointVar).
		Dot("replace").
		Call(
			tsg.NewStatement().New("RegExp").Call(tsg.NewStatement().Lit("/+$")),
			tsg.NewStatement().Lit(""),
		)
	rpcURL := tsg.NewStatement().Const(resultVar).Op("=").Add(trimmedBase).Op("+").Lit("/").Op("+").Lit(prefix)
	cg.Add(rpcURL.Semicolon())
}
