// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package server

import (
	"fmt"
	"log/slog"

	"github.com/flowchartsman/swaggerui"

	"tgp/core/http"
	"tgp/core/i18n"
	"tgp/plugins/swagger/types"
)

func Serve(addr string, swaggerDoc types.Object) (err error) {

	var specBytes []byte
	if specBytes, err = swaggerDoc.ToJSON(); err != nil {
		return fmt.Errorf("%s: %w", i18n.Msg("failed to generate spec"), err)
	}

	s := &server{
		specBytes: specBytes,
	}

	mux := http.NewServeMux()
	mux.Handle("/", swaggerui.Handler(specBytes))

	slog.Info(i18n.Msg("starting swagger server"), slog.String("addr", addr))

	slog.Info(i18n.Msg("swagger server started successfully"), slog.String("addr", addr))

	// Открываем браузер с URL сервера
	browserURL := AddressToURL(addr)
	if err = OpenBrowser(browserURL); err != nil {
		slog.Warn(i18n.Msg("failed to open browser"),
			slog.String("url", browserURL),
			slog.String("error", err.Error()),
		)
		// Не возвращаем ошибку, так как это не критично
	} else {
		slog.Info(i18n.Msg("browser opened successfully"), slog.String("url", browserURL))
	}

	if s.serverID, err = http.ListenAndServe(addr, mux); err != nil {
		slog.Error(i18n.Msg("failed to start swagger server"), slog.String("addr", addr), slog.Any("error", err))
		return fmt.Errorf("%s: %w", i18n.Msg("failed to start server"), err)
	}

	return
}
