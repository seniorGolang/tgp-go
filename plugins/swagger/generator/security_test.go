// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"testing"
)

func TestParseSecurityAnnotations_ProfileWithoutURLs(t *testing.T) {

	security, schemes := parseSecurityAnnotations(`http:bearer:JWT|jwks=/jwks/{sub}/.well-known/keys,oauth2:tokenExchange,oauth2:clientCredentials:jwt-bearer`)
	if len(security) != 1 {
		t.Fatalf("security=%v", security)
	}
	bearer := schemes["http_bearer"]
	if bearer.BearerFormat != "JWT" {
		t.Fatalf("format=%q", bearer.BearerFormat)
	}
	if bearer.JWKSURI != "/jwks/{sub}/.well-known/keys" {
		t.Fatalf("jwks=%q", bearer.JWKSURI)
	}
	ex := schemes["oauth2_tokenexchange"]
	if ex.GrantType != grantTokenExchange {
		t.Fatalf("grant=%q", ex.GrantType)
	}
	if ex.Flows.ClientCredentials.TokenURL != asPlaceholder {
		t.Fatalf("tokenURL=%q", ex.Flows.ClientCredentials.TokenURL)
	}
	cc := schemes["oauth2_clientcredentials"]
	if cc.GrantType != grantJWTBearer {
		t.Fatalf("cc grant=%q", cc.GrantType)
	}
}

func TestParseSecurityAnnotations_RejectsAbsoluteJWKS(t *testing.T) {

	_, schemes := parseSecurityAnnotations(`http:bearer:JWT|jwks=https://auth.example/jwks/{sub}/keys`)
	if len(schemes) != 0 {
		t.Fatalf("schemes=%v", schemes)
	}
}

func TestParseSecurityAnnotations_RejectsOAuth2WithURL(t *testing.T) {

	_, schemes := parseSecurityAnnotations(`oauth2:tokenExchange:https://auth.example/token:-`)
	if len(schemes) != 0 {
		t.Fatalf("schemes=%v", schemes)
	}
}

func TestJWKSPathFromSecurity(t *testing.T) {

	path := JWKSPathFromSecurity(`http:bearer|jwks=/jwks/{sub}/.well-known/keys`)
	if path != "/jwks/{sub}/.well-known/keys" {
		t.Fatalf("path=%q", path)
	}
	if JWKSPathFromSecurity(`http:bearer`) != "" {
		t.Fatal("expected empty path")
	}
}
