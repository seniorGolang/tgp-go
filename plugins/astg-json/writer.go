package main

import (
	"encoding/json"
	"fmt"
	"os"

	"tgp/core/i18n"
	"tgp/internal/common"
	"tgp/internal/model"
)

// writeModel сериализует модель в JSON и пишет её в stdout (пустой out) или в файл.
func writeModel(project *model.Project, out string) (err error) {

	var payload []byte
	if payload, err = json.MarshalIndent(project, "", "  "); err != nil {
		return fmt.Errorf("%s: %w", i18n.Msg("failed to marshal project"), err)
	}

	if out == "" {
		if _, err = os.Stdout.Write(append(payload, '\n')); err != nil {
			return fmt.Errorf("%s: %w", i18n.Msg("failed to write project to stdout"), err)
		}
		return
	}

	path := common.NormalizeWASMPath(out)
	if err = os.WriteFile(path, payload, 0600); err != nil {
		return fmt.Errorf("%s: %w", i18n.Msg("failed to write project file"), err)
	}
	return
}
