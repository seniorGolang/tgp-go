// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"strings"

	"tgp/plugins/swagger/types"
)

const (
	grantJWTBearer     = "urn:ietf:params:oauth:grant-type:jwt-bearer"     //nolint:gosec // G101 — OAuth grant type URN, не секрет
	grantTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange" //nolint:gosec // G101 — OAuth grant type URN, не секрет
	flowTokenExchange  = "tokenexchange"
	asPlaceholder      = "{authorizationServer}"
)

func parseSecurityAnnotations(raw string) (security []types.Security, schemes types.SecuritySchemes) {

	schemes = make(types.SecuritySchemes)

	for _, token := range splitSecurityTokens(raw) {
		name, scheme := buildSecurityScheme(token)
		if name == "" {
			continue
		}

		schemes[name] = scheme
		if scheme.Type == schemeOAuth2 {
			continue
		}
		security = append(security, types.Security{name: {}})
	}

	return
}

func splitSecurityTokens(raw string) (tokens []string) {

	var current string

	for part := range strings.SplitSeq(raw, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		isNew := false
		if index := strings.Index(trimmed, ":"); index > 0 {
			kind := strings.ToLower(strings.TrimSpace(trimmed[:index]))
			switch kind {
			case familyHTTP, "apikey", "openid", schemeOAuth2:
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

	jwksPath := ""
	schemeToken := token
	if before, after, found := strings.Cut(token, "|jwks="); found {
		jwksPath = strings.TrimSpace(after)
		schemeToken = before
		if jwksPath == "" || strings.Contains(jwksPath, "://") {
			return "", types.SecurityScheme{}
		}
		if !strings.HasPrefix(jwksPath, "/") {
			jwksPath = "/" + jwksPath
		}
	}

	parts := strings.Split(schemeToken, ":")
	if len(parts) == 0 {
		return "", types.SecurityScheme{}
	}

	kind := strings.ToLower(strings.TrimSpace(parts[0]))

	switch kind {
	case familyHTTP:
		return buildHTTPScheme(parts, jwksPath)

	case "apikey":
		if jwksPath != "" || len(parts) != 3 {
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

		return "apiKey_" + in + "_" + paramName, types.SecurityScheme{
			Type: "apiKey",
			Name: paramName,
			In:   in,
		}

	case "openid":
		return "", types.SecurityScheme{}

	case schemeOAuth2:
		if jwksPath != "" {
			return "", types.SecurityScheme{}
		}
		return buildOAuth2Scheme(parts)

	default:
		return "", types.SecurityScheme{}
	}
}

func buildHTTPScheme(parts []string, jwksPath string) (name string, scheme types.SecurityScheme) {

	if len(parts) < 2 || len(parts) > 3 {
		return "", types.SecurityScheme{}
	}

	schemeName := strings.ToLower(strings.TrimSpace(parts[1]))
	if schemeName == "" {
		return "", types.SecurityScheme{}
	}

	bearerFormat := ""
	if len(parts) == 3 {
		bearerFormat = strings.TrimSpace(parts[2])
		if bearerFormat == "" {
			return "", types.SecurityScheme{}
		}
	}

	if jwksPath != "" && schemeName != "bearer" {
		return "", types.SecurityScheme{}
	}

	return "http_" + schemeName, types.SecurityScheme{
		Type:         familyHTTP,
		Scheme:       schemeName,
		BearerFormat: bearerFormat,
		JWKSURI:      jwksPath,
	}
}

func buildOAuth2Scheme(parts []string) (name string, scheme types.SecurityScheme) {

	if len(parts) < 2 {
		return "", types.SecurityScheme{}
	}

	flowKind := strings.ToLower(strings.TrimSpace(parts[1]))
	oauthFlows := &types.OAuthFlows{}

	switch flowKind {
	case "clientcredentials":
		grantType := ""
		if len(parts) >= 3 && strings.EqualFold(strings.TrimSpace(parts[2]), "jwt-bearer") {
			grantType = grantJWTBearer
		} else if len(parts) > 2 {
			return "", types.SecurityScheme{}
		}
		oauthFlows.ClientCredentials = buildOAuthFlow("", asPlaceholder, "")
		scheme.GrantType = grantType

	case flowTokenExchange:
		if len(parts) > 2 {
			return "", types.SecurityScheme{}
		}
		oauthFlows.ClientCredentials = buildOAuthFlow("", asPlaceholder, "")
		scheme.GrantType = grantTokenExchange

	case "authorizationcode", "password", "implicit":
		return "", types.SecurityScheme{}

	default:
		return "", types.SecurityScheme{}
	}

	scheme.Type = schemeOAuth2
	scheme.Flows = oauthFlows

	return "oauth2_" + flowKind, scheme
}

func buildOAuthFlow(authURL string, tokenURL string, rawScopes string) (flow *types.OAuthFlow) {

	scopes := make(map[string]string)

	for scope := range strings.SplitSeq(rawScopes, ",") {
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

// JWKSPathFromSecurity returns relative JWKS path template from package security annotation.
func JWKSPathFromSecurity(raw string) (path string) {

	_, schemes := parseSecurityAnnotations(raw)
	if scheme, ok := schemes["http_bearer"]; ok {
		return scheme.JWKSURI
	}
	return ""
}

// HasBearerSecurity reports whether package security enables HTTP bearer.
func HasBearerSecurity(raw string) (ok bool) {

	_, schemes := parseSecurityAnnotations(raw)
	_, ok = schemes["http_bearer"]
	return
}
