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

	securityValue := model.GetAnnotationValue(project, nil, nil, nil, tagSecurity, "")
	if securityValue != "" {
		swaggerDoc.Security, swaggerDoc.Components.SecuritySchemes = parseSecurityAnnotations(securityValue)
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

func parseSecurityAnnotations(raw string) (security []types.Security, schemes types.SecuritySchemes) {

	schemes = make(types.SecuritySchemes)

	for _, token := range splitSecurityTokens(raw) {
		name, scheme := buildSecurityScheme(token)
		if name == "" {
			continue
		}

		schemes[name] = scheme
		security = append(security, types.Security{name: {}})
	}

	return
}

func splitSecurityTokens(raw string) (tokens []string) {

	var current string

	for _, part := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		isNew := false
		if index := strings.Index(trimmed, ":"); index > 0 {
			kind := strings.ToLower(strings.TrimSpace(trimmed[:index]))
			switch kind {
			case "http", "apikey", "openid", "oauth2":
				isNew = true
			}
		}

		switch {
		case isNew:
			if current != "" {
				tokens = append(tokens, current)
			}
			current = trimmed
		case current == "":
			current = trimmed
		default:
			current += "," + trimmed
		}
	}

	if current != "" {
		tokens = append(tokens, current)
	}

	return
}

func buildSecurityScheme(token string) (name string, scheme types.SecurityScheme) {

	parts := strings.Split(token, ":")
	if len(parts) == 0 {
		return "", types.SecurityScheme{}
	}

	kind := strings.ToLower(strings.TrimSpace(parts[0]))

	switch kind {
	case "http":
		if len(parts) != 2 {
			return "", types.SecurityScheme{}
		}
		schemeName := strings.ToLower(strings.TrimSpace(parts[1]))
		if schemeName == "" {
			return "", types.SecurityScheme{}
		}

		return "http_" + schemeName, types.SecurityScheme{
			Type:   "http",
			Scheme: schemeName,
		}

	case "apikey":
		if len(parts) != 3 {
			return "", types.SecurityScheme{}
		}

		in := strings.ToLower(strings.TrimSpace(parts[1]))
		paramName := strings.TrimSpace(parts[2])
		if in != "header" && in != "query" && in != "cookie" {
			return "", types.SecurityScheme{}
		}
		if paramName == "" {
			return "", types.SecurityScheme{}
		}

		schemeName := "apiKey_" + in + "_" + paramName

		return schemeName, types.SecurityScheme{
			Type: "apiKey",
			Name: paramName,
			In:   in,
		}

	case "openid":
		if len(parts) < 2 {
			return "", types.SecurityScheme{}
		}
		url := strings.TrimSpace(strings.Join(parts[1:], ":"))
		if url == "" {
			return "", types.SecurityScheme{}
		}

		return "openId", types.SecurityScheme{
			Type:             "openIdConnect",
			OpenIDConnectURL: url,
		}

	case "oauth2":
		return buildOAuth2Scheme(parts)

	default:
		return "", types.SecurityScheme{}
	}
}

func buildOAuth2Scheme(parts []string) (name string, scheme types.SecurityScheme) {

	if len(parts) < 3 {
		return "", types.SecurityScheme{}
	}

	flowKind := strings.ToLower(strings.TrimSpace(parts[1]))
	rawScopesIndex := 0

	oauthFlows := &types.OAuthFlows{}

	switch flowKind {
	case "clientcredentials":
		tokenURL, scopes, grantType := parseSingleURLFlow(parts)
		if tokenURL == "" {
			return "", types.SecurityScheme{}
		}
		rawScopesIndex = 0

		oauthFlows.ClientCredentials = buildOAuthFlow("", tokenURL, scopes)
		scheme.GrantType = grantType

	case "authorizationcode":
		if len(parts) != 5 {
			return "", types.SecurityScheme{}
		}
		authURL := strings.TrimSpace(parts[2])
		tokenURL := strings.TrimSpace(parts[3])
		if authURL == "" || tokenURL == "" {
			return "", types.SecurityScheme{}
		}
		rawScopesIndex = 4

		oauthFlows.AuthorizationCode = buildOAuthFlow(authURL, tokenURL, parts[rawScopesIndex])

	case "password":
		tokenURL, scopes, grantType := parseSingleURLFlow(parts)
		if tokenURL == "" {
			return "", types.SecurityScheme{}
		}
		rawScopesIndex = 0

		oauthFlows.Password = buildOAuthFlow("", tokenURL, scopes)
		scheme.GrantType = grantType

	case "implicit":
		authURL, scopes, grantType := parseSingleURLFlow(parts)
		if authURL == "" {
			return "", types.SecurityScheme{}
		}
		rawScopesIndex = 0

		oauthFlows.Implicit = buildOAuthFlow(authURL, "", scopes)
		scheme.GrantType = grantType

	default:
		return "", types.SecurityScheme{}
	}

	schemeName := "oauth2_" + flowKind

	scheme.Type = "oauth2"
	scheme.Flows = oauthFlows

	return schemeName, scheme
}

func parseSingleURLFlow(parts []string) (url string, scopes string, grantType string) {

	if len(parts) < 4 {
		return "", "", ""
	}

	lastIndex := len(parts) - 1
	scopesIndex := lastIndex

	possibleGrant := strings.TrimSpace(parts[lastIndex])
	if strings.EqualFold(possibleGrant, "jwt-bearer") {
		grantType = "urn:ietf:params:oauth:grant-type:jwt-bearer"
		scopesIndex--
	}

	if scopesIndex <= 2 {
		return "", "", ""
	}

	urlParts := parts[2:scopesIndex]
	url = strings.TrimSpace(strings.Join(urlParts, ":"))
	if url == "" {
		return "", "", ""
	}

	scopes = strings.TrimSpace(parts[scopesIndex])
	if scopes == "" {
		return "", "", ""
	}

	return url, scopes, grantType
}

func buildOAuthFlow(authURL string, tokenURL string, rawScopes string) (flow *types.OAuthFlow) {

	scopes := make(map[string]string)

	for _, scope := range strings.Split(rawScopes, ",") {
		trimmed := strings.TrimSpace(scope)
		if trimmed == "" {
			continue
		}
		scopes[trimmed] = trimmed
	}

	if authURL == "" && tokenURL == "" && len(scopes) == 0 {
		return nil
	}

	return &types.OAuthFlow{
		AuthorizationURL: authURL,
		TokenURL:         tokenURL,
		Scopes:           scopes,
	}
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

	var hasExplicitTags bool

	if c.Annotations != nil {
		for _, t := range strings.Split(c.Annotations.Value(tagSwaggerTags, ""), ",") {
			trimmed := strings.TrimSpace(t)
			if trimmed == "" {
				continue
			}
			hasExplicitTags = true
			if trimmed == tag {
				return true
			}
		}
	}
	for _, m := range c.Methods {
		if m.Annotations != nil {
			for _, t := range strings.Split(m.Annotations.Value(tagSwaggerTags, ""), ",") {
				trimmed := strings.TrimSpace(t)
				if trimmed == "" {
					continue
				}
				hasExplicitTags = true
				if trimmed == tag {
					return true
				}
			}
		}
	}
	if !hasExplicitTags && strings.TrimSpace(c.Name) == strings.TrimSpace(tag) {
		return true
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
