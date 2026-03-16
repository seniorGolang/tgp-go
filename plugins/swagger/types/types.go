// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package types

import (
	"github.com/goccy/go-json"

	"gopkg.in/yaml.v3"
)

type Object struct {
	OpenAPI    string          `json:"openapi" yaml:"openapi"`
	Info       Info            `json:"info,omitempty" yaml:"info,omitempty"`
	Servers    []Server        `json:"servers,omitempty" yaml:"servers,omitempty"`
	Tags       []Tag           `json:"tags,omitempty" yaml:"tags,omitempty"`
	Schemes    []string        `json:"schemes,omitempty" yaml:"schemes,omitempty"`
	Paths      map[string]Path `json:"paths" yaml:"paths"`
	Components Components      `json:"components,omitempty" yaml:"components,omitempty"`
	Security   []Security      `json:"security,omitempty" yaml:"security,omitempty"`
}

func (o Object) ToJSON() (data []byte, err error) {
	return json.MarshalIndent(o, "", "    ")
}

func (o Object) ToYAML() (data []byte, err error) {
	return yaml.Marshal(o)
}

type Path struct {
	Ref         string     `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Summary     string     `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
	Get         *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	Post        *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	Patch       *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
	Put         *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	Delete      *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	Options     *Operation `json:"options,omitempty" yaml:"options,omitempty"`
}

type Operation struct {
	Tags        []string     `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary     string       `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string       `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Consumes    []string     `json:"consumes,omitempty" yaml:"consumes,omitempty"`
	Produces    []string     `json:"produces,omitempty" yaml:"produces,omitempty"`
	Parameters  []Parameter  `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   Responses    `json:"responses,omitempty" yaml:"responses,omitempty"`
	Deprecated  bool         `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Servers     []Server     `json:"servers,omitempty" yaml:"servers,omitempty"`
	CodeSamples []CodeSample `json:"x-code-samples,omitempty" yaml:"x-code-samples,omitempty"`
}

type CodeSample struct {
	Lang   string `json:"lang" yaml:"lang"`
	Source string `json:"source" yaml:"source"`
}

type Contact struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

type License struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

type Info struct {
	Title          string   `json:"title,omitempty" yaml:"title,omitempty"`
	Description    string   `json:"description,omitempty" yaml:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	License        *License `json:"license,omitempty" yaml:"license,omitempty"`
	Version        string   `json:"version,omitempty" yaml:"version,omitempty"`
}

type ExternalDocs struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	URL         string `json:"url,omitempty" yaml:"url,omitempty"`
}

type Tag struct {
	Name         string       `json:"name,omitempty" yaml:"name,omitempty"`
	Description  string       `json:"description,omitempty" yaml:"description,omitempty"`
	ExternalDocs ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

type Server struct {
	URL         string              `json:"url,omitempty" yaml:"url,omitempty"`
	Description string              `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   map[string]Variable `json:"variables,omitempty" yaml:"variables,omitempty"`
}

type Schemas map[string]Schema

type Properties map[string]Schema

type Components struct {
	Schemas         Schemas         `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	SecuritySchemes SecuritySchemes `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}

type Schema struct {
	Ref         string     `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type        string     `json:"type,omitempty" yaml:"type,omitempty"`
	Format      string     `json:"format,omitempty" yaml:"format,omitempty"`
	Minimum     int        `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum     int        `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	Required    []string   `json:"required,omitempty" yaml:"required,omitempty"`
	Properties  Properties `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items       *Schema    `json:"items,omitempty" yaml:"items,omitempty"`
	Enum        []string   `json:"enum,omitempty" yaml:"enum,omitempty"`
	Nullable    bool       `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	Example     any        `json:"example,omitempty" yaml:"example,omitempty"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`

	OneOf []Schema `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AllOf []Schema `json:"allOf,omitempty" yaml:"allOf,omitempty"`

	AdditionalProperties any `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
}

func (s Schema) IsEmpty() (ok bool) {

	if s.Ref != "" {
		return false
	}
	if s.Format != "" {
		return false
	}
	if len(s.Properties) > 0 {
		return false
	}
	if s.Items != nil {
		return false
	}
	if len(s.AllOf) > 0 || len(s.OneOf) > 0 {
		return false
	}
	return s.Type == "" || (s.Type == "object" && len(s.Properties) == 0)
}

type Variable struct {
	Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default     string   `json:"default,omitempty" yaml:"default,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
}

type Parameter struct {
	Ref         string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	In          string `json:"in,omitempty" yaml:"in,omitempty"`
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Schema      Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type Encoding struct {
	ContentType string `json:"contentType,omitempty" yaml:"contentType,omitempty"`
}

type Media struct {
	Schema   Schema              `json:"schema,omitempty" yaml:"schema,omitempty"`
	Encoding map[string]Encoding `json:"encoding,omitempty" yaml:"encoding,omitempty"`
}

type Content map[string]Media

type Response struct {
	Description string            `json:"description" yaml:"description"`
	Content     Content           `json:"content,omitempty" yaml:"content,omitempty"`
	Headers     map[string]Header `json:"headers,omitempty" yaml:"headers,omitempty"`
}

type Header struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Schema      Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type Responses map[string]Response

type RequestBody struct {
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Content     Content `json:"content,omitempty" yaml:"content,omitempty"`
}

type Security map[string][]string

type SecuritySchemes map[string]SecurityScheme

type SecurityScheme struct {
	Type             string      `json:"type,omitempty" yaml:"type,omitempty"`
	Scheme           string      `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	Name             string      `json:"name,omitempty" yaml:"name,omitempty"`
	In               string      `json:"in,omitempty" yaml:"in,omitempty"`
	OpenIDConnectURL string      `json:"openIdConnectUrl,omitempty" yaml:"openIdConnectUrl,omitempty"`
	Flows            *OAuthFlows `json:"flows,omitempty" yaml:"flows,omitempty"`
	GrantType        string      `json:"x-grant-type,omitempty" yaml:"x-grant-type,omitempty"`
}

type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty" yaml:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty" yaml:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty" yaml:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty" yaml:"authorizationCode,omitempty"`
}

type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty" yaml:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty" yaml:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty" yaml:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
}
