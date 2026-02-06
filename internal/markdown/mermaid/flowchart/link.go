// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package flowchart

import "fmt"

func (f *Flowchart) LinkWithArrowHead(from, to string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s-->%s", from, to))
	return f
}

func (f *Flowchart) LinkWithArrowHeadAndText(from, to, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s-->|\"%s\"|%s", from, text, to))
	return f
}

func (f *Flowchart) OpenLink(from, to string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s --- %s", from, to))
	return f
}

func (f *Flowchart) OpenLinkWithText(from, to, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s---|\"%s\"|%s", from, text, to))
	return f
}

func (f *Flowchart) DottedLink(from, to string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s-.->%s", from, to))
	return f
}

func (f *Flowchart) DottedLinkWithText(from, to, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s-. \"%s\" .-> %s", from, text, to))
	return f
}

func (f *Flowchart) ThickLink(from, to string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s ==> %s", from, to))
	return f
}

func (f *Flowchart) ThickLinkWithText(from, to, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s == \"%s\" ==> %s", from, text, to))
	return f
}

func (f *Flowchart) InvisibleLink(from, to string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s ~~~ %s", from, to))
	return f
}
