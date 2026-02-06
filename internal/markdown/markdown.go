// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package markdown

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"

	"tgp/internal/markdown/internal"
)

type SyntaxHighlight string

const (
	SyntaxHighlightNone         SyntaxHighlight = ""
	SyntaxHighlightText         SyntaxHighlight = "text"
	SyntaxHighlightAPIBlueprint SyntaxHighlight = "markdown"
	SyntaxHighlightShell        SyntaxHighlight = "shell"
	SyntaxHighlightGo           SyntaxHighlight = "go"
	SyntaxHighlightJSON         SyntaxHighlight = "json"
	SyntaxHighlightYAML         SyntaxHighlight = "yaml"
	SyntaxHighlightXML          SyntaxHighlight = "xml"
	SyntaxHighlightHTML         SyntaxHighlight = "html"
	SyntaxHighlightCSS          SyntaxHighlight = "css"
	SyntaxHighlightJavaScript   SyntaxHighlight = "javascript"
	SyntaxHighlightTypeScript   SyntaxHighlight = "typescript"
	SyntaxHighlightSQL          SyntaxHighlight = "sql"
	SyntaxHighlightC            SyntaxHighlight = "c"
	SyntaxHighlightCSharp       SyntaxHighlight = "csharp"
	SyntaxHighlightCPlusPlus    SyntaxHighlight = "cpp"
	SyntaxHighlightJava         SyntaxHighlight = "java"
	SyntaxHighlightKotlin       SyntaxHighlight = "kotlin"
	SyntaxHighlightPHP          SyntaxHighlight = "php"
	SyntaxHighlightPython       SyntaxHighlight = "python"
	SyntaxHighlightRuby         SyntaxHighlight = "ruby"
	SyntaxHighlightSwift        SyntaxHighlight = "swift"
	SyntaxHighlightScala        SyntaxHighlight = "scala"
	SyntaxHighlightRust         SyntaxHighlight = "rust"
	SyntaxHighlightObjectiveC   SyntaxHighlight = "objectivec"
	SyntaxHighlightPerl         SyntaxHighlight = "perl"
	SyntaxHighlightLua          SyntaxHighlight = "lua"
	SyntaxHighlightDart         SyntaxHighlight = "dart"
	SyntaxHighlightClojure      SyntaxHighlight = "clojure"
	SyntaxHighlightGroovy       SyntaxHighlight = "groovy"
	SyntaxHighlightR            SyntaxHighlight = "r"
	SyntaxHighlightHaskell      SyntaxHighlight = "haskell"
	SyntaxHighlightErlang       SyntaxHighlight = "erlang"
	SyntaxHighlightElixir       SyntaxHighlight = "elixir"
	SyntaxHighlightOCaml        SyntaxHighlight = "ocaml"
	SyntaxHighlightJulia        SyntaxHighlight = "julia"
	SyntaxHighlightScheme       SyntaxHighlight = "scheme"
	SyntaxHighlightFSharp       SyntaxHighlight = "fsharp"
	SyntaxHighlightCoffeeScript SyntaxHighlight = "coffeescript"
	SyntaxHighlightVBNet        SyntaxHighlight = "vbnet"
	SyntaxHighlightTeX          SyntaxHighlight = "tex"
	SyntaxHighlightDiff         SyntaxHighlight = "diff"
	SyntaxHighlightApache       SyntaxHighlight = "apache"
	SyntaxHighlightDockerfile   SyntaxHighlight = "dockerfile"
	SyntaxHighlightMermaid      SyntaxHighlight = "mermaid"
)

type TableOfContentsDepth int

const (
	TableOfContentsDepthH1 TableOfContentsDepth = 1
	TableOfContentsDepthH2 TableOfContentsDepth = 2
	TableOfContentsDepthH3 TableOfContentsDepth = 3
	TableOfContentsDepthH4 TableOfContentsDepth = 4
	TableOfContentsDepthH5 TableOfContentsDepth = 5
	TableOfContentsDepthH6 TableOfContentsDepth = 6
)

const (
	TableOfContentsMarkerBegin = "<!-- BEGIN_TOC -->"
	TableOfContentsMarkerEnd   = "<!-- END_TOC -->"
)

type TableOfContentsOptions struct {
	MinDepth TableOfContentsDepth
	MaxDepth TableOfContentsDepth
}

type headerInfo struct {
	level TableOfContentsDepth
	text  string
}

type Markdown struct {
	body        []string
	dest        io.Writer
	err         error
	headers     []headerInfo
	tocOptions  *TableOfContentsOptions
	tocInserted bool
}

func NewMarkdown(w io.Writer) *Markdown {
	return &Markdown{
		body:    []string{},
		dest:    w,
		headers: []headerInfo{},
	}
}

func (m *Markdown) String() string {
	content := strings.Join(m.body, internal.LineFeed())

	if m.tocInserted && m.tocOptions != nil {
		tocContent := m.generateTableOfContents()
		if len(tocContent) > 0 {
			tocText := strings.Join(tocContent, internal.LineFeed())
			placeholder := TableOfContentsMarkerBegin + internal.LineFeed() + TableOfContentsMarkerEnd
			replacement := TableOfContentsMarkerBegin + internal.LineFeed() + tocText + internal.LineFeed() + TableOfContentsMarkerEnd
			content = strings.ReplaceAll(content, placeholder, replacement)
		}
	}

	return content
}

func (m *Markdown) Error() error {
	return m.err
}

func (m *Markdown) PlainText(text string) *Markdown {
	m.body = append(m.body, text)
	return m
}

func (m *Markdown) PlainTextf(format string, args ...any) *Markdown {
	return m.PlainText(fmt.Sprintf(format, args...))
}

func (m *Markdown) Build() error {
	if _, err := fmt.Fprint(m.dest, m.String()); err != nil {
		if m.err != nil {
			return fmt.Errorf("failed to write markdown text: %w: %s", err, m.err.Error()) //nolint:wrapcheck
		}
		return fmt.Errorf("failed to write markdown text: %w", err)
	}
	return m.err
}

func (m *Markdown) H1(text string) *Markdown {
	m.headers = append(m.headers, headerInfo{level: TableOfContentsDepthH1, text: text})
	m.body = append(m.body, fmt.Sprintf("# %s", text))
	return m
}

func (m *Markdown) H1f(format string, args ...any) *Markdown {
	return m.H1(fmt.Sprintf(format, args...))
}

func (m *Markdown) H2(text string) *Markdown {
	m.headers = append(m.headers, headerInfo{level: TableOfContentsDepthH2, text: text})
	m.body = append(m.body, fmt.Sprintf("## %s", text))
	return m
}

func (m *Markdown) H2f(format string, args ...any) *Markdown {
	return m.H2(fmt.Sprintf(format, args...))
}

func (m *Markdown) H3(text string) *Markdown {
	m.headers = append(m.headers, headerInfo{level: TableOfContentsDepthH3, text: text})
	m.body = append(m.body, fmt.Sprintf("### %s", text))
	return m
}

func (m *Markdown) H3f(format string, args ...any) *Markdown {
	return m.H3(fmt.Sprintf(format, args...))
}

func (m *Markdown) H4(text string) *Markdown {
	m.headers = append(m.headers, headerInfo{level: TableOfContentsDepthH4, text: text})
	m.body = append(m.body, fmt.Sprintf("#### %s", text))
	return m
}

func (m *Markdown) H4f(format string, args ...any) *Markdown {
	return m.H4(fmt.Sprintf(format, args...))
}

func (m *Markdown) H5(text string) *Markdown {
	m.headers = append(m.headers, headerInfo{level: TableOfContentsDepthH5, text: text})
	m.body = append(m.body, fmt.Sprintf("##### %s", text))
	return m
}

func (m *Markdown) H5f(format string, args ...any) *Markdown {
	return m.H5(fmt.Sprintf(format, args...))
}

func (m *Markdown) H6(text string) *Markdown {
	m.headers = append(m.headers, headerInfo{level: TableOfContentsDepthH6, text: text})
	m.body = append(m.body, fmt.Sprintf("###### %s", text))
	return m
}

func (m *Markdown) H6f(format string, args ...any) *Markdown {
	return m.H6(fmt.Sprintf(format, args...))
}

// Example:
//
//	markdown.NewMarkdown(os.Stdout).
//	   TableOfContents(markdown.TableOfContentsDepthH3).  // Table of contents will be placed here
func (m *Markdown) TableOfContents(maxDepth TableOfContentsDepth) *Markdown {
	return m.TableOfContentsWithRange(TableOfContentsDepthH1, maxDepth)
}

// Example:
//
//	markdown.NewMarkdown(os.Stdout).
func (m *Markdown) TableOfContentsWithRange(minDepth, maxDepth TableOfContentsDepth) *Markdown {
	if m.tocInserted {
		if m.err == nil {
			m.err = errors.New("table of contents has already been generated")
		}
		return m
	}

	if minDepth < TableOfContentsDepthH1 || minDepth > TableOfContentsDepthH6 {
		if m.err == nil {
			m.err = fmt.Errorf("invalid minDepth: %d (must be between 1 and 6)", minDepth)
		}
		return m
	}

	if maxDepth < TableOfContentsDepthH1 || maxDepth > TableOfContentsDepthH6 {
		if m.err == nil {
			m.err = fmt.Errorf("invalid maxDepth: %d (must be between 1 and 6)", maxDepth)
		}
		return m
	}

	if minDepth > maxDepth {
		if m.err == nil {
			m.err = fmt.Errorf("minDepth (%d) cannot be greater than maxDepth (%d)", minDepth, maxDepth)
		}
		return m
	}

	m.tocOptions = &TableOfContentsOptions{
		MinDepth: minDepth,
		MaxDepth: maxDepth,
	}
	m.tocInserted = true

	m.body = append(m.body, TableOfContentsMarkerBegin)
	m.body = append(m.body, TableOfContentsMarkerEnd)
	m.body = append(m.body, "")

	return m
}

func (m *Markdown) generateTableOfContents() []string {
	if m.tocOptions == nil || len(m.headers) == 0 {
		return []string{}
	}

	tocLines := make([]string, 0, len(m.headers))
	minIndent := int(m.tocOptions.MinDepth)

	for _, header := range m.headers {
		if header.level < m.tocOptions.MinDepth || header.level > m.tocOptions.MaxDepth {
			continue
		}

		indent := strings.Repeat("  ", int(header.level)-minIndent)

		anchor := strings.ToLower(strings.ReplaceAll(header.text, " ", "-"))
		anchor = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				return r
			}
			return -1
		}, anchor)

		tocLines = append(tocLines, fmt.Sprintf("%s- [%s](#%s)", indent, header.text, anchor))
	}

	return tocLines
}

func (m *Markdown) Details(summary, text string) *Markdown {
	m.body = append(
		m.body,
		fmt.Sprintf("<details><summary>%s</summary>%s%s%s</details>",
			summary, internal.LineFeed(), text, internal.LineFeed()))
	return m
}

func (m *Markdown) Detailsf(summary, format string, args ...any) *Markdown {
	return m.Details(summary, fmt.Sprintf(format, args...))
}

func (m *Markdown) BulletList(text ...string) *Markdown {
	for _, v := range text {
		m.body = append(m.body, fmt.Sprintf("- %s", v))
	}
	return m
}

func (m *Markdown) OrderedList(text ...string) *Markdown {
	for i, v := range text {
		m.body = append(m.body, fmt.Sprintf("%d. %s", i+1, v))
	}
	return m
}

type CheckBoxSet struct {
	Checked bool
	Text    string
}

func (m *Markdown) CheckBox(set []CheckBoxSet) *Markdown {
	for _, v := range set {
		if v.Checked {
			m.body = append(m.body, fmt.Sprintf("- [x] %s", v.Text))
		} else {
			m.body = append(m.body, fmt.Sprintf("- [ ] %s", v.Text))
		}
	}
	return m
}

func (m *Markdown) Blockquote(text string) *Markdown {
	lines := strings.Split(text, internal.LineFeed())
	for _, line := range lines {
		m.body = append(m.body, fmt.Sprintf("> %s", line))
	}
	return m
}

func (m *Markdown) CodeBlocks(lang SyntaxHighlight, text string) *Markdown {
	m.body = append(m.body,
		fmt.Sprintf("```%s%s%s%s```", lang, internal.LineFeed(), text, internal.LineFeed()))
	return m
}

func (m *Markdown) HorizontalRule() *Markdown {
	m.body = append(m.body, "---")
	return m
}

type TableAlignment int

const (
	AlignDefault TableAlignment = iota
	AlignLeft
	AlignCenter
	AlignRight
)

type TableSet struct {
	Header    []string
	Rows      [][]string
	Alignment []TableAlignment
}

func (t *TableSet) ValidateColumns() error {
	headerColumns := len(t.Header)
	for _, record := range t.Rows {
		if len(record) != headerColumns {
			return ErrMismatchColumn
		}
	}
	return nil
}

func (m *Markdown) Table(t TableSet) *Markdown {
	if err := t.ValidateColumns(); err != nil {
		if m.err != nil {
			m.err = fmt.Errorf("failed to validate columns: %w: %s", err, m.err.Error()) //nolint:wrapcheck
		} else {
			m.err = fmt.Errorf("failed to validate columns: %w", err)
		}
		return m
	}

	if len(t.Header) == 0 {
		return m
	}

	var buf strings.Builder

	buf.WriteString("|")
	for _, header := range t.Header {
		buf.WriteString(" ")
		buf.WriteString(header)
		buf.WriteString(" |")
	}
	buf.WriteString(internal.LineFeed())

	buf.WriteString("|")
	for i := 0; i < len(t.Header); i++ {
		align := AlignDefault
		if i < len(t.Alignment) {
			align = t.Alignment[i]
		}

		switch align {
		case AlignDefault:
			buf.WriteString("---------|")
		case AlignLeft:
			buf.WriteString(":--------|")
		case AlignCenter:
			buf.WriteString(":-------:|")
		case AlignRight:
			buf.WriteString("--------:|")
		}
	}
	buf.WriteString(internal.LineFeed())

	for _, row := range t.Rows {
		buf.WriteString("|")
		for _, cell := range row {
			buf.WriteString(" ")
			buf.WriteString(cell)
			buf.WriteString(" |")
		}
		buf.WriteString(internal.LineFeed())
	}

	m.body = append(m.body, buf.String())
	return m
}

type TableOptions struct {
	// AutoWrapText is whether to wrap the text automatically.
	AutoWrapText bool
	// AutoFormatHeaders is whether to format the header automatically.
	AutoFormatHeaders bool
}

func (m *Markdown) CustomTable(t TableSet, options TableOptions) *Markdown {
	if err := t.ValidateColumns(); err != nil {
		// NOTE: If go version is 1.20, use errors.Join
		if m.err != nil {
			m.err = fmt.Errorf("failed to validate columns: %w: %s", err, m.err.Error()) //nolint:wrapcheck
		} else {
			m.err = fmt.Errorf("failed to validate columns: %w", err)
		}
	}

	buf := &strings.Builder{}
	table := tablewriter.NewTable(
		buf,
		tablewriter.WithRenderer(
			renderer.NewBlueprint(
				tw.Rendition{
					Symbols: tw.NewSymbolCustom("Markdown").
						WithHeaderLeft("|").
						WithHeaderRight("|").
						WithColumn("|").
						WithMidLeft("|").
						WithMidRight("|").
						WithCenter("|"),
					Borders: tw.Border{
						Left:   tw.On,
						Top:    tw.Off,
						Right:  tw.On,
						Bottom: tw.Off,
					},
				},
			),
		),
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Formatting: tw.CellFormatting{
					AutoFormat: func() tw.State {
						if options.AutoFormatHeaders {
							return tw.Success
						}
						return tw.Fail
					}(),
				},
			},
			Row: tw.CellConfig{
				Formatting: tw.CellFormatting{
					AutoWrap: func() int {
						if options.AutoWrapText {
							return tw.WrapNormal
						}
						return tw.WrapNone
					}(),
					AutoFormat: func() tw.State {
						if options.AutoFormatHeaders {
							return tw.Success
						}
						return tw.Fail
					}(),
				},

				Alignment: tw.CellAlignment{Global: tw.AlignNone},
			},
		}),
	)

	table.Header(t.Header)
	if err := table.Bulk(t.Rows); err != nil {
		m.err = errors.Join(m.err, fmt.Errorf("failed to add rows to table: %w", err))
		return m
	}
	// This is so if the user wants to change the table settings they can
	if err := table.Render(); err != nil {
		m.err = errors.Join(m.err, fmt.Errorf("failed to render table: %w", err))
		return m
	}

	m.body = append(m.body, buf.String())
	return m
}

func (m *Markdown) LF() *Markdown {
	m.body = append(m.body, "  ")
	return m
}
