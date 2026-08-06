// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"strings"

	"tgp/internal/model"
)

const tagSecurity = "security"

func (r *ClientRenderer) securityRaw() (raw string) {

	return model.GetAnnotationValue(r.project, nil, nil, nil, tagSecurity, "")
}

func (r *ClientRenderer) hasTokenPackage() (ok bool) {

	raw := r.securityRaw()
	if raw == "" {
		return false
	}
	return strings.Contains(raw, "http:bearer") || strings.Contains(raw, "oauth2:")
}

func (r *ClientRenderer) hasApp2AppAuth() (ok bool) {

	raw := r.securityRaw()
	if raw == "" {
		return false
	}
	return strings.Contains(raw, "oauth2:tokenExchange") && strings.Contains(raw, "jwt-bearer")
}
