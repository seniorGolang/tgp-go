// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package arch

import "fmt"

func (a *Architecture) Group(groupID string, icon Icon, title string) *Architecture {
	a.body = append(a.body, fmt.Sprintf("    group %s(%s)[%s]", groupID, icon, title))
	return a
}

func (a *Architecture) GroupInParentGroup(groupID string, icon Icon, title, parentGroupID string) *Architecture {
	a.body = append(a.body, fmt.Sprintf("    group %s(%s)[%s] in %s", groupID, icon, title, parentGroupID))
	return a
}
