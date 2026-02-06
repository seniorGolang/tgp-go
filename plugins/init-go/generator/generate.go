package generator

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"tgp/core/i18n"
)

//go:embed templates
var templates embed.FS

// Generate создаёт базовый проект в outDir (уже разрешённый путь к директории вывода).
func Generate(outDir string, moduleName string, jsonRPC string, rest string) (err error) {

	if err = ensureEmptyOut(outDir); err != nil {
		return fmt.Errorf("%s: %w", i18n.Msg("out directory"), err)
	}
	if strings.TrimSpace(jsonRPC) == "" && strings.TrimSpace(rest) == "" {
		jsonRPC = "some"
	}
	p := newParams(moduleName, jsonRPC, rest)

	var tmpl *template.Template
	if tmpl, err = template.ParseFS(templates, "templates/*.tmpl"); err != nil {
		return fmt.Errorf("%s: %w", i18n.Msg("parse templates"), err)
	}

	if err = renderFile(tmpl, "gitignore.tmpl", filepath.Join(outDir, fileGitignore), p); err != nil {
		return
	}
	if err = renderFile(tmpl, "gomod.tmpl", filepath.Join(outDir, fileGoMod), p); err != nil {
		return
	}
	if err = renderFile(tmpl, "tg.tmpl", filepath.Join(outDir, dirContracts, fileTg), p); err != nil {
		return
	}
	for _, iface := range p.JsonRPCIfaces {
		data := ContractIfaceData{
			Module:        p.Module,
			HTTPPrefix:    p.HTTPPrefix,
			Kind:          "jsonrpc",
			PublicName:    iface.PublicName,
			EntityName:    iface.PublicName,
			EntitySnake:   lowerFirst(iface.PublicName),
			NeedDtoImport: false,
		}
		if err = renderFile(tmpl, "contract_iface.tmpl", filepath.Join(outDir, dirContracts, iface.FileBase+".go"), data); err != nil {
			return
		}
	}
	for _, iface := range p.RestIfaces {
		data := ContractIfaceData{
			Module:        p.Module,
			HTTPPrefix:    p.HTTPPrefix,
			Kind:          "rest",
			PublicName:    iface.PublicName,
			EntityName:    iface.PublicName,
			EntitySnake:   lowerFirst(iface.PublicName),
			NeedDtoImport: true,
		}
		if err = renderFile(tmpl, "contract_iface.tmpl", filepath.Join(outDir, dirContracts, iface.FileBase+".go"), data); err != nil {
			return
		}
	}
	if p.HasREST {
		if err = renderFile(tmpl, "some.tmpl", filepath.Join(outDir, dirContracts, dirDTO, fileSome), p); err != nil {
			return
		}
	}
	if err = renderFile(tmpl, "config.tmpl", filepath.Join(outDir, dirInternal, dirConfig, fileService), p); err != nil {
		return
	}
	if err = renderFile(tmpl, "version.tmpl", filepath.Join(outDir, dirInternal, dirTransport, fileVersion), p); err != nil {
		return
	}
	for _, iface := range p.JsonRPCIfaces {
		svcDir := filepath.Join(outDir, dirInternal, dirServices, iface.FileBase)
		svcData := struct {
			Module      string
			PackageName string
		}{Module: p.Module, PackageName: iface.FileBase}
		if err = renderFile(tmpl, "service_contract.tmpl", filepath.Join(svcDir, fileService), svcData); err != nil {
			return
		}
		doData := struct {
			Module      string
			PackageName string
		}{Module: p.Module, PackageName: iface.FileBase}
		if err = renderFile(tmpl, "do.tmpl", filepath.Join(svcDir, "do.go"), doData); err != nil {
			return
		}
	}
	for _, iface := range p.RestIfaces {
		svcDir := filepath.Join(outDir, dirInternal, dirServices, iface.FileBase)
		svcData := struct {
			Module      string
			PackageName string
		}{Module: p.Module, PackageName: iface.FileBase}
		if err = renderFile(tmpl, "service_contract.tmpl", filepath.Join(svcDir, fileService), svcData); err != nil {
			return
		}
		crudData := struct {
			Module      string
			PackageName string
		}{Module: p.Module, PackageName: iface.FileBase}
		if err = renderFile(tmpl, "crud.tmpl", filepath.Join(svcDir, "crud.go"), crudData); err != nil {
			return
		}
	}
	cmdData := struct {
		Module          string
		ServiceName     string
		ServicePackages []InterfaceSpec
	}{
		Module:          p.Module,
		ServiceName:     p.ServiceName,
		ServicePackages: append(append([]InterfaceSpec{}, p.JsonRPCIfaces...), p.RestIfaces...),
	}
	if err = renderFile(tmpl, "cmd.tmpl", filepath.Join(outDir, dirCmd, p.CmdPackage, fileCmd), cmdData); err != nil {
		return
	}
	errsDir := filepath.Join(outDir, dirPkg, dirErrs)
	if err = renderFile(tmpl, "errs_basic.tmpl", filepath.Join(errsDir, fileBasic), p); err != nil {
		return
	}
	if err = renderFile(tmpl, "errs_type.tmpl", filepath.Join(errsDir, fileType), p); err != nil {
		return
	}
	if err = renderFile(tmpl, "errs_utils.tmpl", filepath.Join(errsDir, fileUtils), p); err != nil {
		return
	}
	if err = renderFile(tmpl, "errs_decode.tmpl", filepath.Join(errsDir, fileDecode), p); err != nil {
		return
	}
	if err = runGoGenerateCMD(outDir); err != nil {
		return fmt.Errorf("%s: %w", i18n.Msg("run go commands"), err)
	}
	if err = runGoTidyCMD(outDir); err != nil {
		return fmt.Errorf("%s: %w", i18n.Msg("run go commands"), err)
	}
	return
}
