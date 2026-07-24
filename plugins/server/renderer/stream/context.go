// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package stream

type contextKey int

// KeyOverlay — context key для map[string]string HTTP overlay (headers/cookies/query/path) на WS session.
const KeyOverlay contextKey = 1
