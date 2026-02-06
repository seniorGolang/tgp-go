package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

func renderFile(tmpl *template.Template, templateName string, filePath string, data any) (err error) {

	dir := filepath.Dir(filePath)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	var buf bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		return fmt.Errorf("execute %s: %w", templateName, err)
	}
	if err = os.WriteFile(filePath, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}
	return
}
