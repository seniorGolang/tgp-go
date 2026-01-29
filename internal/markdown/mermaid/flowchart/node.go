// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package flowchart

import "fmt"

func (f *Flowchart) Node(name string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s", name))
	return f
}

func (f *Flowchart) NodeWithText(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s[\"%s\"]", name, text))
	return f
}

func (f *Flowchart) NodeWithMarkdown(name, markdownText string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s[\"`%s`\"]", name, markdownText))
	return f
}

func (f *Flowchart) NodeWithNewLines(name, textWithNewLines string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s[\"`%s`\"]", name, textWithNewLines))
	return f
}

func (f *Flowchart) RoundEdgesNode(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s(\"%s\")", name, text))
	return f
}

func (f *Flowchart) StadiumNode(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s([\"%s\"])", name, text))
	return f
}

func (f *Flowchart) SubroutineNode(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s[[\"%s\"]]", name, text))
	return f
}

func (f *Flowchart) CylindricalNode(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s[(\"%s\")]", name, text))
	return f
}

func (f *Flowchart) DatabaseNode(name, text string) *Flowchart {
	return f.CylindricalNode(name, text)
}

func (f *Flowchart) CircleNode(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s((\"%s\"))", name, text))
	return f
}

func (f *Flowchart) AsymmetricNode(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s>\"%s\"]", name, text))
	return f
}

func (f *Flowchart) RhombusNode(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s{\"%s\"}", name, text))
	return f
}

func (f *Flowchart) HexagonNode(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s{{\"%s\"}}", name, text))
	return f
}

func (f *Flowchart) ParallelogramNode(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s[/\"%s\"/]", name, text))
	return f
}

func (f *Flowchart) ParallelogramAltNode(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s[\\\"%s\"\\]", name, text))
	return f
}

func (f *Flowchart) TrapezoidNode(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s[/\"%s\"\\]", name, text))
	return f
}

func (f *Flowchart) TrapezoidAltNode(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s[\\\"%s\"/]", name, text))
	return f
}

func (f *Flowchart) DoubleCircleNode(name, text string) *Flowchart {
	f.body = append(f.body, fmt.Sprintf("    %s(((\"%s\")))", name, text))
	return f
}
