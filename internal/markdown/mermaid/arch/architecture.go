// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

// Package arch is mermaid architecture diagram builder.
// The building blocks of an architecture are groups, services, edges, and junctions.
// The arch package incorporates beta features of Mermaid, so the specifications are subject to significant changes.
package arch

import (
	"fmt"
	"io"
	"strings"

	"tgp/internal/markdown/internal"
)

type Architecture struct {
	err    error
	body   []string
	dest   io.Writer
	config *config
}

func NewArchitecture(w io.Writer, opts ...Option) (a *Architecture) {

	c := newConfig()

	for _, opt := range opts {
		opt(c)
	}

	return &Architecture{
		body:   []string{"architecture-beta"},
		dest:   w,
		config: c,
	}
}

func (a *Architecture) String() (s string) {
	return strings.Join(a.body, internal.LineFeed())
}

func (a *Architecture) Build() (err error) {

	if _, writeErr := a.dest.Write([]byte(a.String())); writeErr != nil {
		if a.err != nil {
			return fmt.Errorf("failed to write: %w: %s", writeErr, a.err.Error()) //nolint:wrapcheck
		}
		return fmt.Errorf("failed to write: %w", writeErr)
	}
	return a.err
}

func (a *Architecture) Error() (err error) {
	return a.err
}

func (a *Architecture) LF() *Architecture {

	a.body = append(a.body, "")
	return a
}
