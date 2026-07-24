// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package kafka

import (
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"tgp/internal/generated"
)

//go:embed templates/*.go.tmpl
var templates embed.FS

// WriteCodecOptions задаёт состав генерируемых встроенных кодеков.
type WriteCodecOptions struct {
	PackageJSON    string
	IncludeMsgpack bool
	IncludeCBOR    bool
	IncludeYAML    bool
	IncludeXML     bool
}

// WriteCodec записывает runtime встроенных кодеков.
func WriteCodec(outDir string, pkgName string, opts WriteCodecOptions) (err error) {

	replacements := map[string]string{
		"{{JSON_IMPORT}}":          `"encoding/json"`,
		"{{MSGPACK}}":              "",
		"{{CBOR}}":                 "",
		"{{YAML}}":                 "",
		"{{XML}}":                  "",
		"{{MSGPACK_REGISTRATION}}": "",
		"{{CBOR_REGISTRATION}}":    "",
		"{{YAML_REGISTRATION}}":    "",
		"{{XML_REGISTRATION}}":     "",
	}
	if opts.PackageJSON != "" {
		replacements["{{JSON_IMPORT}}"] = `json "` + opts.PackageJSON + `"`
	}
	if opts.IncludeMsgpack {
		replacements["{{MSGPACK}}"] = "\t\"github.com/vmihailenco/msgpack/v5\"\n"
		replacements["{{MSGPACK_REGISTRATION}}"] = "\tcodecs[\"msgpack\"] = jsonCodec{marshal: msgpack.Marshal, unmarshal: msgpack.Unmarshal}\n"
	}
	if opts.IncludeCBOR {
		replacements["{{CBOR}}"] = "\t\"github.com/fxamacker/cbor/v2\"\n"
		replacements["{{CBOR_REGISTRATION}}"] = "\tcodecs[\"cbor\"] = jsonCodec{marshal: cbor.Marshal, unmarshal: cbor.Unmarshal}\n"
	}
	if opts.IncludeYAML {
		replacements["{{YAML}}"] = "\t\"gopkg.in/yaml.v3\"\n"
		replacements["{{YAML_REGISTRATION}}"] = "\tcodecs[\"yaml\"] = jsonCodec{marshal: yaml.Marshal, unmarshal: yaml.Unmarshal}\n"
	}
	if opts.IncludeXML {
		replacements["{{XML}}"] = "\t\"encoding/xml\"\n"
		replacements["{{XML_REGISTRATION}}"] = "\tcodecs[\"xml\"] = jsonCodec{marshal: xml.Marshal, unmarshal: xml.Unmarshal}\n"
	}
	return writeTemplate(outDir, pkgName, "codec.go", replacements)
}

// WriteProduce записывает runtime отправки сообщений.
func WriteProduce(outDir string, pkgName string) (err error) {

	return writeTemplate(outDir, pkgName, "produce.go", nil)
}

// WriteRecord записывает runtime обёрток Kafka-записей.
func WriteRecord(outDir string, pkgName string) (err error) {

	return writeTemplate(outDir, pkgName, "record.go", nil)
}

// WritePoll записывает runtime опроса и диспетчеризации топиков.
func WritePoll(outDir string, pkgName string) (err error) {

	return writeTemplate(outDir, pkgName, "poll.go", nil)
}

// WriteSecurity записывает runtime построения SASL-механизма.
func WriteSecurity(outDir string, pkgName string) (err error) {

	return writeTemplate(outDir, pkgName, "security.go", nil)
}

func writeTemplate(outDir string, pkgName string, name string, replacements map[string]string) (err error) {

	if pkgName == "" {
		return fmt.Errorf("kafka runtime package name is required")
	}
	var source []byte
	if source, err = templates.ReadFile("templates/" + name + ".tmpl"); err != nil {
		return fmt.Errorf("read kafka runtime template %q: %w", name, err)
	}
	content := strings.Replace(string(source), "package kafkaRUNTIME", "package "+pkgName, 1)
	for from, to := range replacements {
		content = strings.ReplaceAll(content, from, to)
	}
	content = generated.ByToolGatewayComment + content
	var formatted []byte
	if formatted, err = format.Source([]byte(content)); err != nil {
		return fmt.Errorf("format kafka runtime %q: %w", name, err)
	}
	if err = os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create kafka runtime directory: %w", err)
	}
	if err = os.WriteFile(filepath.Join(outDir, name), formatted, 0o644); err != nil {
		return fmt.Errorf("write kafka runtime %q: %w", name, err)
	}
	return nil
}
