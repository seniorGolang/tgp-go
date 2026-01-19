// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package server

import (
	"fmt"
	"log/slog"

	"github.com/flowchartsman/swaggerui"

	"tgp/core/http"
	"tgp/core/i18n"
	"tgp/plugins/swagger/types"
)

// Serve запускает HTTP сервер с Swagger UI на указанном адресе.
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

	var serverID uint64
	if serverID, err = http.ListenAndServe(addr, mux); err != nil {
		slog.Error(i18n.Msg("failed to start swagger server"), slog.String("addr", addr), slog.Any("error", err))
		return fmt.Errorf("%s: %w", i18n.Msg("failed to start server"), err)
	}

	s.listenerID = serverID

	slog.Info(i18n.Msg("swagger server started successfully"), slog.String("addr", addr))

	return
}
