// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"tgp/internal/model"
	"tgp/internal/tags"
)

func (r *contractRenderer) isInlineField(v *model.Variable, methodTags tags.DocTags) (ok bool) {

	return tags.HasJSONInline(methodTags, v.Name)
}

func (r *contractRenderer) requestStructFieldName(method *model.Method, v *model.Variable) (s string) {

	if r.isInlineField(v, method.Annotations) {
		return model.TypeNameFromTypeID(r.project, v.TypeID)
	}

	return toCamel(v.Name)
}

func (r *contractRenderer) responseStructFieldName(method *model.Method, ret *model.Variable) (s string) {

	results := resultsWithoutError(method)
	if len(results) == 1 && model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHttpEnableInlineSingle) {
		if model.TypeIsEmbeddable(r.project, ret.TypeID) {
			return model.TypeNameFromTypeID(r.project, ret.TypeID)
		}

		return toCamel(ret.Name)
	}

	return toCamel(ret.Name)
}
