// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"strings"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func filterDocsComments(docs []string) (filtered []string) {

	for _, d := range docs {
		if !strings.Contains(d, "@tg") {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func resolveDescValue(value string) (s string) {

	return value
}

func descriptionFromDocsAndTags(docs []string, annotations tags.DocTags) (out string) {

	if annotations != nil {
		if v := annotations.Value(tagDesc, ""); v != "" {
			return resolveDescValue(v)
		}
	}
	filtered := filterDocsComments(docs)
	if len(filtered) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(filtered, "  \n"))
}

func descriptionFromVariable(v *model.Variable) (out string) {

	if v == nil {
		return ""
	}
	return descriptionFromDocsAndTags(v.Docs, v.Annotations)
}

func descriptionFromMethod(m *model.Method) (out string) {

	if m == nil {
		return ""
	}
	return descriptionFromDocsAndTags(m.Docs, m.Annotations)
}

func descriptionFromType(typ *model.Type) (out string) {

	if typ == nil || len(typ.Docs) == 0 {
		return ""
	}
	parsed := tags.ParseTags(typ.Docs)
	return descriptionFromDocsAndTags(typ.Docs, parsed)
}

func descriptionFromStructField(field *model.StructField) (out string) {

	if field == nil {
		return ""
	}
	parsed := tags.ParseTags(field.Docs)
	return descriptionFromDocsAndTags(field.Docs, parsed)
}

func descriptionFromProject(project *model.Project) (out string) {

	if project == nil {
		return ""
	}
	return descriptionFromDocsAndTags(project.Docs, project.Annotations)
}

func requestBodyDescription(m *model.Method) (out string) {

	if m == nil {
		return ""
	}
	if m.Annotations != nil {
		if v := m.Annotations.Value(tagRequestBodyDesc, ""); v != "" {
			return resolveDescValue(v)
		}
	}
	return descriptionFromMethod(m)
}
