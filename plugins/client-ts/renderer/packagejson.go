// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-json"

	"tgp/internal/model"
)

func (r *ClientRenderer) RenderPackageJSON() (err error) {

	if r.packageJSONPath == "" {
		return nil
	}

	name := model.GetAnnotationValue(r.project, nil, nil, nil, tagNpmName, "")
	if name == "" {
		return fmt.Errorf("package-json requires @tg npmName annotation")
	}

	packageDir := filepath.Dir(r.packageJSONPath)
	mainPath, err := npmRelativePath(packageDir, filepath.Join(r.outDir, "dist", "client.js"))
	if err != nil {
		return fmt.Errorf("package-json main path: %w", err)
	}
	typesPath, err := npmRelativePath(packageDir, filepath.Join(r.outDir, "dist", "client.d.ts"))
	if err != nil {
		return fmt.Errorf("package-json types path: %w", err)
	}
	distDir, err := npmRelativePath(packageDir, filepath.Join(r.outDir, "dist"))
	if err != nil {
		return fmt.Errorf("package-json dist path: %w", err)
	}
	readmePath, err := npmRelativePath(packageDir, filepath.Join(r.outDir, "readme.md"))
	if err != nil {
		return fmt.Errorf("package-json readme path: %w", err)
	}
	distDir = strings.TrimPrefix(distDir, "./")
	readmePath = strings.TrimPrefix(readmePath, "./")

	pkg := map[string]any{
		"name":    name,
		"version": npmVersion(model.GetAnnotationValue(r.project, nil, nil, nil, tagVersion, "0.0.0")),
		"type":    "module",
		"main":    mainPath,
		"types":   typesPath,
		"exports": map[string]any{
			".": map[string]any{
				"types":  typesPath,
				"import": mainPath,
			},
		},
		"files": []string{distDir, readmePath},
		"scripts": map[string]string{
			"build":          "tsc",
			"prepublishOnly": "npm run build",
		},
		"devDependencies": map[string]string{
			"typescript": "^5.4.0",
		},
	}

	if desc := model.GetAnnotationValue(r.project, nil, nil, nil, tagDesc, ""); desc != "" {
		pkg["description"] = desc
	}
	if license := model.GetAnnotationValue(r.project, nil, nil, nil, tagLicense, ""); license != "" {
		pkg["license"] = license
	}
	if author := model.GetAnnotationValue(r.project, nil, nil, nil, tagAuthor, ""); author != "" {
		pkg["author"] = author
	}
	if model.GetAnnotationValueBool(r.project, nil, nil, nil, tagNpmPrivate, false) {
		pkg["private"] = true
	}
	if registry := model.GetAnnotationValue(r.project, nil, nil, nil, tagNpmRegistry, ""); registry != "" {
		pkg["publishConfig"] = map[string]string{"registry": registry}
	}

	var jsonData []byte
	if jsonData, err = json.MarshalIndent(pkg, "", "  "); err != nil {
		return fmt.Errorf("marshal package.json: %w", err)
	}
	jsonData = append(jsonData, '\n')
	if err = os.MkdirAll(packageDir, 0700); err != nil {
		return fmt.Errorf("create package.json directory: %w", err)
	}
	if err = os.WriteFile(r.packageJSONPath, jsonData, 0600); err != nil {
		return fmt.Errorf("write package.json: %w", err)
	}
	return nil
}

func npmRelativePath(packageDir string, target string) (rel string, err error) {

	var relPath string
	if relPath, err = filepath.Rel(packageDir, target); err != nil {
		return "", err
	}
	return "./" + filepath.ToSlash(relPath), nil
}

func npmVersion(raw string) (version string) {

	version = strings.TrimSpace(raw)
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	if version == "" {
		return "0.0.0"
	}
	return version
}
