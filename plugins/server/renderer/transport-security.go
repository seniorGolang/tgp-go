// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"strings"

	"tgp/internal/model"
)

const tagSecurity = "security"

type jwksSegment struct {
	Literal string
	Param   string
	Index   int
}

func (r *transportRenderer) securityJWKSPath() (pathTemplate string) {

	raw := model.GetAnnotationValue(r.project, nil, nil, nil, tagSecurity, "")
	if _, after, found := strings.Cut(raw, "|jwks="); found {
		pathTemplate = strings.TrimSpace(after)
		if cut := strings.IndexAny(pathTemplate, ",|"); cut >= 0 {
			pathTemplate = pathTemplate[:cut]
		}
		pathTemplate = strings.TrimSpace(pathTemplate)
		if strings.Contains(pathTemplate, "://") {
			return ""
		}
		if pathTemplate != "" && !strings.HasPrefix(pathTemplate, "/") {
			pathTemplate = "/" + pathTemplate
		}
	}
	return pathTemplate
}

func parseJWKSPath(template string) (params []string, segments []jwksSegment) {

	if template == "" {
		return
	}
	paramIndex := make(map[string]int)
	for token := range strings.SplitSeq(template, "/") {
		if token == "" {
			continue
		}
		if name, ok := strings.CutPrefix(token, ":"); ok && name != "" {
			index, exists := paramIndex[name]
			if !exists {
				index = len(params)
				paramIndex[name] = index
				params = append(params, name)
			}
			segments = append(segments, jwksSegment{Param: name, Index: index})
			continue
		}
		segments = append(segments, jwksSegment{Literal: token})
	}
	return
}

func (r *transportRenderer) bearerTemplateData() (data *pkgTemplateData) {

	data = newPkgTemplateData()
	data.JWKSPath = r.securityJWKSPath()
	data.JWKSParams, data.JWKSSegments = parseJWKSPath(data.JWKSPath)
	return
}

func (r *transportRenderer) RenderTransportSecurity() (err error) {

	return
}
