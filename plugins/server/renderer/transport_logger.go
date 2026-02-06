// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path"
)

func (r *transportRenderer) RenderTransportLogger() error {

	loggerPath := path.Join(r.outDir, "logger.go")

	if err := os.Remove(loggerPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
