// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package arch

import "fmt"

func (a *Architecture) Junction(junctionID string) *Architecture {
	a.body = append(a.body, fmt.Sprintf("    junction %s", junctionID))
	return a
}

func (a *Architecture) JunctionsInParent(junctionID, parentGroupID string) *Architecture {
	a.body = append(a.body, fmt.Sprintf("    junction %s in %s", junctionID, parentGroupID))
	return a
}
