// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

// Package flowchart provides a simple way to create flowcharts in mermaid syntax.
package flowchart

import (
	"fmt"
	"io"
	"strings"

	"tgp/internal/markdown/internal"
)

type Flowchart struct {
	err    error
	body   []string
	dest   io.Writer
	config *config
}

func NewFlowchart(w io.Writer, opts ...Option) (f *Flowchart) {

	c := newConfig()

	for _, opt := range opts {
		opt(c)
	}

	lines := []string{}
	if strings.TrimSpace(c.title) != noTitle {
		lines = append(lines, "---")
		lines = append(lines, fmt.Sprintf("title: %s", c.title))
		lines = append(lines, "---")
	}
	lines = append(lines, fmt.Sprintf("flowchart %s", c.oriental.string()))

	return &Flowchart{
		body:   lines,
		dest:   w,
		config: c,
	}
}

func (f *Flowchart) String() (s string) {
	return strings.Join(f.body, internal.LineFeed())
}

func (f *Flowchart) Build() (err error) {

	if _, writeErr := fmt.Fprint(f.dest, f.String()); writeErr != nil {
		if f.err != nil {
			return fmt.Errorf("failed to write: %w: %s", writeErr, f.err.Error()) //nolint:wrapcheck
		}
		return fmt.Errorf("failed to write: %w", writeErr)
	}
	return f.err
}
