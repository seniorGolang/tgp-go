// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	_ "embed"
	"os"
	"path"

	"tgp/internal/generated"
)

//go:embed templates/headers.ts
var headersTS []byte

//go:embed templates/headers-plain.ts
var headersPlainTS []byte

func (r *ClientRenderer) RenderHeaders() (err error) {

	template := headersPlainTS
	if r.clientIdentity {
		template = headersTS
	}
	content := append([]byte(generated.ByToolGatewayComment+"\n"), template...)
	err = os.WriteFile(path.Join(r.outDir, "headers.ts"), content, 0600)
	return
}
