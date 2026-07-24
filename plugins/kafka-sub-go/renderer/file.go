// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"

	"tgp/plugins/kafka-sub-go/goimports"
)

func writeSource(outDir string, name string, source string) (err error) {

	var formatted []byte
	if formatted, err = format.Source([]byte(source)); err != nil {
		return fmt.Errorf("format %s: %w", name, err)
	}
	if err = os.MkdirAll(outDir, 0o755); err != nil {
		return
	}
	filePath := filepath.Join(outDir, name)
	if err = os.WriteFile(filePath, formatted, 0o644); err != nil {
		return
	}
	var runner goimports.Runner
	if runner, err = goimports.NewFromFile(filePath); err != nil {
		return
	}
	return runner.Run(goimports.GetModulePath(filePath))
}
