// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	_ "embed"
	"os"
	"path"

	"tgp/internal/generated"
)

//go:embed templates/identity.ts
var identityTS []byte

func (r *ClientRenderer) RenderIdentity() (err error) {

	if !r.clientIdentity {
		return
	}

	content := append([]byte(generated.ByToolGatewayComment+"\n"), identityTS...)
	err = os.WriteFile(path.Join(r.outDir, "identity.ts"), content, 0600)
	return
}
