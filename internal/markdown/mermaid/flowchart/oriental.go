// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package flowchart

type oriental string

const (
	// tb is a top to bottom oriental.
	tb oriental = "TB"
	// td is a top down oriental, same as top to Bottom.
	td oriental = "TD"
	// bt is a bottom to top oriental.
	bt oriental = "BT"
	// rl is a right to left oriental.
	rl oriental = "RL"
	// lr is a left to right oriental.
	lr oriental = "LR"
)

func (o oriental) string() string {
	return string(o)
}
