// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package arch

import "fmt"

func (a *Architecture) Service(serviceID string, icon Icon, title string) *Architecture {
	a.body = append(a.body, fmt.Sprintf("    service %s(%s)[%s]", serviceID, icon, title))
	return a
}

func (a *Architecture) ServiceInGroup(serviceID string, icon Icon, title, groupID string) *Architecture {
	a.body = append(a.body, fmt.Sprintf("    service %s(%s)[%s] in %s", serviceID, icon, title, groupID))
	return a
}
