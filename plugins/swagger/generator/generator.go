// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"strings"

	"tgp/internal/model"
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

// GenerateDoc генерирует swagger документацию на основе проекта.
func GenerateDoc(project *model.Project, ifaces ...string) (swaggerDoc types.Object) {

	gen := newGenerator(project)

	swaggerDoc.OpenAPI = openAPIVersion
	swaggerDoc.Info.Title = project.Annotations.Value(tagTitle, project.ModulePath)
	swaggerDoc.Info.Version = project.Annotations.Value(tagAppVersion, defaultVersion)
	swaggerDoc.Info.Description = project.Annotations.Value(tagDesc, "")
	swaggerDoc.Paths = make(map[string]types.Path)

	var servers string
	if servers = project.Annotations.Value(tagServers, ""); servers != "" {
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
	if security = project.Annotations.Value(tagSecurity, ""); security != "" {
		securityList := strings.Split(security, "|")
		for _, sec := range securityList {
			if strings.EqualFold(sec, bearerSecuritySchema) {
				swaggerDoc.Security = append(swaggerDoc.Security, types.Security{
					BearerAuth: []interface{}{},
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

	swaggerDoc.Paths = gen.generatePaths(project.Contracts, ifaces)
	swaggerDoc.Components.Schemas = gen.schemas

	return
}
