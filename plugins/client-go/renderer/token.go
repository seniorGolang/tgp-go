// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
)

func (r *ClientRenderer) RenderToken() (err error) {

	if !r.hasTokenPackage() {
		return
	}
	if err = r.pkgRenderTo("token", r.outDir, newPkgTemplateData()); err != nil {
		return fmt.Errorf("render token package: %w", err)
	}
	return
}
