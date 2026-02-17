// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"sort"
	"strings"

	"tgp/internal/model"
	"tgp/internal/validate"
	"tgp/plugins/swagger/types"
)

type generator struct {
	project    *model.Project
	schemas    types.Schemas
	knownCount map[string]int
	knownTypes map[string]types.Schema
}

func newGenerator(project *model.Project) (gen *generator) {
	return &generator{
		project:    project,
		schemas:    make(types.Schemas),
		knownCount: make(map[string]int),
		knownTypes: make(map[string]types.Schema),
	}
}

func GenerateDoc(project *model.Project, ifaces ...string) (swaggerDoc types.Object, err error) {

	if err = validate.Project(project); err != nil {
		return swaggerDoc, fmt.Errorf("invalid project: %w", err)
	}

	for _, contract := range project.Contracts {
		if err = validate.Contract(contract, project); err != nil {
			return swaggerDoc, fmt.Errorf("validate contract %q: %w", contract.Name, err)
		}
	}

	gen := newGenerator(project)

	swaggerDoc.OpenAPI = openAPIVersion
	swaggerDoc.Info.Title = model.GetAnnotationValue(project, nil, nil, nil, tagTitle, project.ModulePath)
	swaggerDoc.Info.Version = model.GetAnnotationValue(project, nil, nil, nil, tagAppVersion, defaultVersion)
	swaggerDoc.Info.Description = descriptionFromProject(project)
	swaggerDoc.Paths = make(map[string]types.Path)

	var servers string
	if servers = model.GetAnnotationValue(project, nil, nil, nil, tagServers, ""); servers != "" {
		serverList := strings.Split(servers, "|")
		for _, server := range serverList {
			serverValues := strings.Split(server, ";")
			serverURL := serverValues[0]
			var serverDesc string
			if len(serverValues) > 1 {
				serverDesc = serverValues[1]
			}
			swaggerDoc.Servers = append(swaggerDoc.Servers, types.Server{
				URL:         serverURL,
				Description: serverDesc,
			})
		}
	}

	var security string
	if security = model.GetAnnotationValue(project, nil, nil, nil, tagSecurity, ""); security != "" {
		securityList := strings.Split(security, "|")
		for _, sec := range securityList {
			if strings.EqualFold(sec, bearerSecuritySchema) {
				swaggerDoc.Security = append(swaggerDoc.Security, types.Security{
					BearerAuth: []any{},
				})
				swaggerDoc.Components.SecuritySchemes = &types.SecuritySchemes{
					BearerAuth: types.BearerAuth{
						Type:   "http",
						Scheme: sec,
					},
				}
			}
		}
	}

	paths := gen.generatePaths(project.Contracts, ifaces)
	swaggerDoc.Paths = paths

	tagOrder := collectTagOrder(paths)
	sort.Strings(tagOrder)
	tagDescs := buildTagDescriptions(project.Contracts, tagOrder)
	for _, name := range tagOrder {
		swaggerDoc.Tags = append(swaggerDoc.Tags, types.Tag{Name: name, Description: tagDescs[name]})
	}

	swaggerDoc.Components.Schemas = gen.schemas

	return
}

func collectTagOrder(paths map[string]types.Path) (order []string) {

	seen := make(map[string]bool)
	for _, p := range paths {
		for _, op := range []*types.Operation{p.Get, p.Post, p.Put, p.Patch, p.Delete, p.Options} {
			if op == nil {
				continue
			}
			for _, t := range op.Tags {
				t = strings.TrimSpace(t)
				if t != "" && !seen[t] {
					seen[t] = true
					order = append(order, t)
				}
			}
		}
	}
	return order
}

func contractHasTag(c *model.Contract, tag string) (has bool) {

	if c.Annotations != nil {
		for _, t := range strings.Split(c.Annotations.Value(tagSwaggerTags, ""), ",") {
			if strings.TrimSpace(t) == tag {
				return true
			}
		}
	}
	for _, m := range c.Methods {
		if m.Annotations != nil {
			for _, t := range strings.Split(m.Annotations.Value(tagSwaggerTags, ""), ",") {
				if strings.TrimSpace(t) == tag {
					return true
				}
			}
		}
	}
	return false
}

func buildTagDescriptions(contracts []*model.Contract, tagOrder []string) (tagToDesc map[string]string) {

	tagToDesc = make(map[string]string)
	for _, tag := range tagOrder {
		if _, ok := tagToDesc[tag]; ok {
			continue
		}
		for _, c := range contracts {
			if !contractHasTag(c, tag) {
				continue
			}
			desc := ""
			if c.Annotations != nil {
				desc = c.Annotations.Value(tagTagDesc+"."+tag, "")
			}
			if desc == "" {
				desc = descriptionFromDocsAndTags(c.Docs, c.Annotations)
			}
			tagToDesc[tag] = desc
			break
		}
	}
	return tagToDesc
}
