// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"slices"

	"tgp/internal/model"
	"tgp/internal/tags"
	"tgp/plugins/client-ts/tsg"
)

type tsInterfaceFieldOptions struct {
	seen       map[string]bool
	skipNames  map[string]struct{}
	isArgument bool
}

func (r *ClientRenderer) jsonName(field *model.StructField) (value string, inline bool) {

	return tags.StructFieldJSONName(field.Tags, field.Name)
}

func (r *ClientRenderer) appendStructFieldTSInterface(grp *tsg.Group, field *model.StructField, pkgPath string, opts tsInterfaceFieldOptions) {

	if field == nil || grp == nil {
		return
	}

	jsonName, inline := r.jsonName(field)
	if jsonName == "-" {
		return
	}

	if inline && model.StructFieldEmbedded(r.project, field) {
		embedded, ok := r.project.Types[field.TypeID]
		if !ok || embedded.Kind != model.TypeKindStruct {
			return
		}
		embeddedPkg := pkgPath
		if embedded.ImportPkgPath != "" {
			embeddedPkg = embedded.ImportPkgPath
		}
		for _, embeddedField := range embedded.StructFields {
			r.appendStructFieldTSInterface(grp, embeddedField, embeddedPkg, opts)
		}
		return
	}

	if jsonName == "" {
		return
	}

	if opts.skipNames != nil {
		if _, skip := opts.skipNames[jsonName]; skip {
			return
		}
	}

	if opts.seen != nil {
		if opts.seen[jsonName] {
			return
		}
		opts.seen[jsonName] = true
	}

	fieldVar := &model.Variable{
		TypeRef: field.TypeRef,
		Name:    field.Name,
	}
	fieldTags := parseTagsFromDocs(field.Docs)
	typeStr := r.walkVariable(field.Name, pkgPath, fieldVar, fieldTags, opts.isArgument).typeLink()

	fieldStmt := tsg.NewStatement()
	fieldStmt.ID(jsonName)
	if r.structFieldTSOptional(field) {
		fieldStmt.Optional()
	}
	fieldStmt.Colon()
	fieldStmt.Add(tsg.TypeFromString(typeStr))
	fieldStmt.Semicolon()
	grp.Add(fieldStmt)
}

func (r *ClientRenderer) structFieldTSOptional(field *model.StructField) (ok bool) {

	if field == nil {
		return false
	}

	if field.NumberOfPointers > 0 {
		return true
	}

	if tagValues, found := field.Tags["json"]; found {
		if slices.Contains(tagValues, "omitempty") {
			return true
		}
	}

	return false
}

func (r *ClientRenderer) collectStructReadmeFields(structType *model.Type) (fields []*model.StructField) {

	if structType == nil {
		return nil
	}

	for _, field := range structType.StructFields {
		_, inline := r.jsonName(field)
		if inline && model.StructFieldEmbedded(r.project, field) {
			embedded, ok := r.project.Types[field.TypeID]
			if !ok || embedded.Kind != model.TypeKindStruct {
				fields = append(fields, field)
				continue
			}
			fields = append(fields, embedded.StructFields...)
			continue
		}
		fields = append(fields, field)
	}

	return
}
