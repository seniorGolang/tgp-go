// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package cache

import (
	"log/slog"

	"tgp/core/i18n"
	"tgp/plugins/astg/marker"
)

func GetProjectID(rootDir string) (id string, err error) {

	id, err = marker.ProjectID(rootDir)
	if err != nil {
		slog.Debug(i18n.Msg("failed to compute project ID"), slog.String("error", err.Error()), slog.String("rootDir", rootDir))
	}
	return id, err
}
