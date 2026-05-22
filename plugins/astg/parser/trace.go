// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"log/slog"
	"runtime"
	"runtime/debug"
)

const traceLogKey = "astgTrace"

func traceRecover(stage string) {

	if recovered := recover(); recovered != nil {
		slog.Error("astg panic",
			slog.String(traceLogKey, stage),
			slog.String("panic", fmt.Sprintf("%v", recovered)),
			slog.String("stack", string(debug.Stack())),
		)
		panic(recovered)
	}
}

func traceBegin(stage string, args ...any) {

	attrs := append([]any{slog.String(traceLogKey, stage)}, args...)
	slog.Debug("astg trace begin", attrs...)
}

func traceEnd(stage string, args ...any) {

	attrs := append([]any{slog.String(traceLogKey, stage)}, args...)
	slog.Debug("astg trace end", attrs...)
}

func traceStep(stage string, args ...any) {

	attrs := append([]any{slog.String(traceLogKey, stage)}, args...)
	slog.Debug("astg trace", attrs...)
}

func traceRuntime() {

	traceStep("runtime", slog.String("version", runtime.Version()))
}
