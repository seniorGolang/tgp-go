// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package descref

import (
	"strings"

	"tgp/internal/model"
	"tgp/internal/tags"
)

const filePrefix = "file:"

// Формат значения: file:path или file:path#section.
func ResolveFileRef(rootDir string, value string) (resolved string, err error) {

	if !strings.HasPrefix(value, filePrefix) {
		return value, nil
	}
	rest := value[len(filePrefix):]
	var relPath string
	var section string
	if idx := strings.Index(rest, "#"); idx >= 0 {
		relPath = strings.TrimSpace(rest[:idx])
		section = strings.TrimSpace(rest[idx+1:])
	} else {
		relPath = strings.TrimSpace(rest)
	}
	return readFileContent(rootDir, relPath, section)
}

// ResolveFileRefsInProject заменяет во всех Annotations значения с префиксом file: на содержимое файлов.
func ResolveFileRefsInProject(project *model.Project, rootDir string) {

	if project == nil {
		return
	}
	resolveDocTags(project.Annotations, rootDir)
	for _, c := range project.Contracts {
		resolveContract(c, rootDir)
	}
	for _, t := range project.Types {
		resolveType(t, rootDir)
	}
}

func resolveContract(c *model.Contract, rootDir string) {

	if c == nil {
		return
	}
	resolveDocTags(c.Annotations, rootDir)
	for _, m := range c.Methods {
		resolveMethod(m, rootDir)
	}
}

func resolveMethod(m *model.Method, rootDir string) {

	if m == nil {
		return
	}
	resolveDocTags(m.Annotations, rootDir)
	for _, v := range m.Args {
		resolveVariable(v, rootDir)
	}
	for _, v := range m.Results {
		resolveVariable(v, rootDir)
	}
}

func resolveVariable(v *model.Variable, rootDir string) {

	if v == nil {
		return
	}
	resolveDocTags(v.Annotations, rootDir)
}

func resolveType(t *model.Type, rootDir string) {

	if t == nil {
		return
	}
	for _, v := range t.EmbeddedInterfaces {
		resolveVariable(v, rootDir)
	}
	for _, v := range t.FunctionArgs {
		resolveVariable(v, rootDir)
	}
	for _, v := range t.FunctionResults {
		resolveVariable(v, rootDir)
	}
	for _, f := range t.InterfaceMethods {
		resolveFunction(f, rootDir)
	}
}

func resolveFunction(f *model.Function, rootDir string) {

	if f == nil {
		return
	}
	for _, v := range f.Args {
		resolveVariable(v, rootDir)
	}
	for _, v := range f.Results {
		resolveVariable(v, rootDir)
	}
}

func resolveDocTags(ann tags.DocTags, rootDir string) {

	if ann == nil {
		return
	}

	for k, v := range ann {
		if !strings.HasPrefix(v, filePrefix) {
			continue
		}
		resolved, err := ResolveFileRef(rootDir, v)
		if err != nil {
			continue
		}
		ann[k] = resolved
	}
}
