package content

import "strings"

const (
	KindJSON    = "json"
	KindForm    = "form"
	KindXML     = "xml"
	KindMsgpack = "msgpack"
	KindCBOR    = "cbor"
	KindYAML    = "yaml"
)

var mimeToKind = map[string]string{
	"application/json":                  KindJSON,
	"application/x-www-form-urlencoded": KindForm,
	"application/xml":                   KindXML,
	"text/xml":                          KindXML,
	"application/msgpack":               KindMsgpack,
	"application/x-msgpack":             KindMsgpack,
	"application/cbor":                  KindCBOR,
	"application/yaml":                  KindYAML,
	"application/x-yaml":                KindYAML,
	"text/yaml":                         KindYAML,
}

var kindToCanonicalMIME = map[string]string{
	KindJSON:    "application/json",
	KindForm:    "application/x-www-form-urlencoded",
	KindXML:     "application/xml",
	KindMsgpack: "application/msgpack",
	KindCBOR:    "application/cbor",
	KindYAML:    "application/x-yaml",
}

func Kind(mime string) string {

	mime = strings.TrimSpace(strings.Split(mime, ";")[0])
	if k, ok := mimeToKind[mime]; ok {
		return k
	}
	return KindJSON
}

func CanonicalMIME(kind string) string {

	if m, ok := kindToCanonicalMIME[kind]; ok {
		return m
	}
	return kindToCanonicalMIME[KindJSON]
}
