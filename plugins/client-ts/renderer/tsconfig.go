// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"tgp/internal/generated"
)

func (r *ClientRenderer) RenderTSConfig() (err error) {

	outDir := r.outDir

	// Структура для tsconfig.json
	compilerOptions := map[string]any{
		"target":                           targetES2020,
		"module":                           targetES2020,
		"lib":                              []string{targetES2020, "DOM"},
		"moduleResolution":                 "bundler",
		"strict":                           true,
		"esModuleInterop":                  true,
		"skipLibCheck":                     true,
		"resolveJsonModule":                true,
		"allowSyntheticDefaultImports":     true,
		"forceConsistentCasingInFileNames": true,
		"declaration":                      true,
		"declarationMap":                   true,
		"sourceMap":                        true,
		"noUnusedLocals":                   true,
		"noUnusedParameters":               true,
	}
	if r.emitDist {
		compilerOptions["outDir"] = "dist"
		compilerOptions["rootDir"] = "."
	}
	tsConfig := map[string]any{
		"compilerOptions": compilerOptions,
		"include": []string{
			"**/*.ts",
		},
		"exclude": []string{
			"node_modules",
		},
	}

	var jsonData []byte
	if jsonData, err = json.MarshalIndent(tsConfig, "", "  "); err != nil {
		return fmt.Errorf("failed to marshal tsconfig.json: %w", err)
	}

	fileContent := generated.ByToolGatewayComment + string(jsonData) + "\n"

	tsConfigPath := path.Join(outDir, "tsconfig.json")
	if err = os.WriteFile(tsConfigPath, []byte(fileContent), 0o600); err != nil {
		return fmt.Errorf("failed to write tsconfig.json: %w", err)
	}

	return
}
