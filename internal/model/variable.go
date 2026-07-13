// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import "tgp/internal/tags"

// EffectiveVariable возвращает копию variable с объединёнными аннотациями:
// method.Annotations.Sub(variable.Name), перекрытые variable.Annotations.
func EffectiveVariable(method *Method, variable *Variable) (effective *Variable) {

	if variable == nil {
		return nil
	}
	if method == nil || len(method.Annotations) == 0 {
		return variable
	}

	subAnnotations := method.Annotations.Sub(variable.Name)
	if len(subAnnotations) == 0 {
		return variable
	}

	mergedAnnotations := make(tags.DocTags, len(subAnnotations)+len(variable.Annotations))
	for key, value := range subAnnotations {
		mergedAnnotations[key] = value
	}
	for key, value := range variable.Annotations {
		mergedAnnotations[key] = value
	}

	variableCopy := *variable
	variableCopy.Annotations = mergedAnnotations
	return &variableCopy
}

func methodVariableAnnotations(method *Method, variable *Variable) (annotations tags.DocTags) {

	if method == nil || variable == nil || len(method.Annotations) == 0 {
		return nil
	}
	return method.Annotations.Sub(variable.Name)
}
