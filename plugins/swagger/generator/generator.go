// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
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

	if err = validate.ValidateProject(project); err != nil {
		return swaggerDoc, fmt.Errorf("invalid project: %w", err)
	}

	for _, contract := range project.Contracts {
		if err = validate.ValidateContract(contract, project); err != nil {
			return swaggerDoc, fmt.Errorf("validate contract %q: %w", contract.Name, err)
		}
	}

	gen := newGenerator(project)

	swaggerDoc.OpenAPI = openAPIVersion
	swaggerDoc.Info.Title = model.GetAnnotationValue(project, nil, nil, nil, tagTitle, project.ModulePath)
	swaggerDoc.Info.Version = model.GetAnnotationValue(project, nil, nil, nil, tagAppVersion, defaultVersion)
	swaggerDoc.Info.Description = model.GetAnnotationValue(project, nil, nil, nil, tagDesc, "")
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

	swaggerDoc.Paths = gen.generatePaths(project.Contracts, ifaces)
	swaggerDoc.Components.Schemas = gen.schemas

	return
}
