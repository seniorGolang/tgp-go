// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"tgp/internal/model"
)

type ClientRenderer struct {
	project                  *model.Project
	outDir                   string
	contract                 *model.Contract
	knownTypes               map[string]int
	typeDefTs                map[string]typeDefTs
	typeAnchorsSet           map[string]bool
	needParseFormValueHelper bool
}

func NewClientRenderer(project *model.Project, outDir string) (r *ClientRenderer) {
	return &ClientRenderer{
		project:    project,
		outDir:     outDir,
		knownTypes: make(map[string]int),
		typeDefTs:  make(map[string]typeDefTs),
	}
}

func (r *ClientRenderer) HasJsonRPC() (ok bool) {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			return true
		}
	}
	return false
}

func (r *ClientRenderer) HasHTTP() (ok bool) {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
			return true
		}
	}
	return false
}

func (r *ClientRenderer) ContractKeys() (keys []string) {

	keys = make([]string, 0, len(r.project.Contracts))
	for _, c := range r.project.Contracts {
		keys = append(keys, c.Name)
	}
	sort.Strings(keys)
	return
}

func (r *ClientRenderer) FindContract(name string) (out *model.Contract) {

	for _, c := range r.project.Contracts {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func (r *ClientRenderer) isTypeFromCurrentProject(importPkgPath string) (ok bool) {

	// Если ImportPkgPath начинается с ModulePath проекта, это тип из текущего проекта
	if r.project.ModulePath != "" && strings.HasPrefix(importPkgPath, r.project.ModulePath) {
		return true
	}
	return false
}

func (r *ClientRenderer) tsFileName(contract *model.Contract) (s string) {

	name := contract.Name
	if len(name) == 0 {
		return ""
	}

	result := make([]rune, 0, len(name)*2)
	for i, c := range name {
		if i > 0 {
			prev := rune(name[i-1])
			if unicode.IsUpper(c) && unicode.IsLower(prev) {
				result = append(result, '-')
			}
			if unicode.IsUpper(c) && unicode.IsUpper(prev) {
				if i+1 < len(name) && unicode.IsLower(rune(name[i+1])) {
					result = append(result, '-')
				}
			}
		}
		result = append(result, unicode.ToLower(c))
	}
	return string(result)
}

func (r *ClientRenderer) fileNameToMethodName(fileName string) (methodName string) {

	if fileName == "" {
		return ""
	}
	parts := strings.FieldsFunc(fileName, func(c rune) bool { return c == '-' || c == '_' })
	var b strings.Builder
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 {
			b.WriteString(part)
			continue
		}
		if len(part) > 0 {
			b.WriteString(strings.ToUpper(string(part[0])))
			b.WriteString(part[1:])
		}
	}
	return b.String()
}

func (r *ClientRenderer) lcName(s string) (out string) {

	if len(s) == 0 {
		return ""
	}
	return toLowerCamel(s)
}

func (r *ClientRenderer) requestTypeName(contract *model.Contract, method *model.Method) (s string) {
	return fmt.Sprintf("Request%s%s", contract.Name, method.Name)
}

func (r *ClientRenderer) responseTypeName(contract *model.Contract, method *model.Method) (s string) {
	return fmt.Sprintf("Response%s%s", contract.Name, method.Name)
}

func toLowerCamel(s string) (out string) {

	if s == "" {
		return s
	}
	isAllUpper := true
	for _, v := range s {
		if v >= 'a' && v <= 'z' {
			isAllUpper = false
			break
		}
	}
	if isAllUpper {
		return s
	}
	if len(s) > 0 {
		first := rune(s[0])
		if first >= 'A' && first <= 'Z' {
			s = strings.ToLower(string(first)) + s[1:]
		}
	}
	parts := strings.Split(s, "_")
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(string(parts[i][0])) + parts[i][1:]
		}
	}
	out = result
	return
}
