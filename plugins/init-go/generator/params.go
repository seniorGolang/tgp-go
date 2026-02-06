package generator

import (
	"strings"
	"unicode"
)

// InterfaceSpec задаёт один интерфейс (JSON-RPC или REST): публичное имя и базовое имя файла.
type InterfaceSpec struct {
	PublicName string // PascalCase для типа интерфейса (Some, SiteNova)
	FileBase   string // snake_case для имени файла (some.go, site_nova.go)
}

// ContractIfaceData — данные для рендера одного файла контракта (один интерфейс).
type ContractIfaceData struct {
	Module        string
	HTTPPrefix    string
	Kind          string // "jsonrpc" или "rest"
	PublicName    string
	EntityName    string
	EntitySnake   string
	NeedDtoImport bool
}

type params struct {
	Module         string
	JsonRPCIfaces  []InterfaceSpec
	RestIfaces     []InterfaceSpec
	EntityName     string
	EntitySnake    string
	HasJSONRPC     bool
	HasREST        bool
	ServiceName    string
	CmdPackage     string
	SwaggerMinimal string
	HTTPPrefix     string
}

func newParams(moduleName string, jsonRPCRaw string, restRaw string) (p params) {

	p.Module = moduleName
	p.JsonRPCIfaces = parseAndNormalizeInterfaces(jsonRPCRaw)
	p.RestIfaces = parseAndNormalizeInterfaces(restRaw)
	p.HasJSONRPC = len(p.JsonRPCIfaces) > 0
	p.HasREST = len(p.RestIfaces) > 0
	p.EntityName = entityNameDefault
	if p.HasJSONRPC {
		p.EntityName = p.JsonRPCIfaces[0].PublicName
	} else if p.HasREST {
		p.EntityName = p.RestIfaces[0].PublicName
	}
	p.EntitySnake = lowerFirst(p.EntityName)
	p.ServiceName = moduleName + "-service"
	p.CmdPackage = moduleName
	p.SwaggerMinimal = `{"openapi":"3.0.0","info":{"title":"` + moduleName + ` API","version":"v1.0.0"},"paths":{}}`
	p.HTTPPrefix = httpPrefixDefault
	return
}

// parseAndNormalizeInterfaces разбивает строку по запятой, очищает и возвращает срез уникальных InterfaceSpec.
func parseAndNormalizeInterfaces(raw string) (out []InterfaceSpec) {

	seen := make(map[string]struct{})
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		public := toPublicName(s)
		base := publicNameToFileBase(public)
		key := base
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, InterfaceSpec{PublicName: public, FileBase: base})
	}
	return out
}

// toPublicName приводит строку к публичному имени типа в Go (PascalCase).
// Примеры: "some" -> "Some", "siteNova" -> "SiteNova", "site_nova" -> "SiteNova", "SOME" -> "Some".
func toPublicName(s string) (out string) {

	if s == "" {
		return ""
	}
	if strings.Contains(s, "_") {
		var out []string
		for _, part := range strings.Split(s, "_") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			out = append(out, toExport(part))
		}
		return strings.Join(out, "")
	}
	if isAllUpper(s) {
		return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
	}
	return toExport(s)
}

func isAllUpper(s string) bool {

	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsUpper(r) {
			return false
		}
	}
	return len(s) > 0 && unicode.IsLetter(rune(s[0]))
}

// publicNameToFileBase переводит публичное имя в базовое имя файла Go (snake_case).
// Примеры: "Some" -> "some", "SiteNova" -> "site_nova".
func publicNameToFileBase(publicName string) (out string) {

	if publicName == "" {
		return ""
	}
	var b strings.Builder
	for i, r := range publicName {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func toExport(s string) (out string) {

	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func lowerFirst(s string) (out string) {

	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}
