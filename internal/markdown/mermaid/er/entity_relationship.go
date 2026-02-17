// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

// Package er is mermaid entity relationship diagram builder.
package er

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"tgp/internal/markdown/internal"
)

type Diagram struct {
	err      error
	body     []string
	dest     io.Writer
	config   *config
	entities sync.Map
}

func NewDiagram(w io.Writer, opts ...Option) (d *Diagram) {

	c := newConfig()

	for _, opt := range opts {
		opt(c)
	}

	return &Diagram{
		body:     []string{"erDiagram"},
		dest:     w,
		config:   c,
		entities: sync.Map{},
	}
}

func (d *Diagram) String() (s string) {

	s = strings.Join(d.body, internal.LineFeed())
	s += internal.LineFeed()

	entities := make([]Entity, 0)
	d.entities.Range(func(_, value any) bool {
		e, ok := value.(Entity)
		if !ok {
			return false
		}
		entities = append(entities, e)
		return true
	})

	sort.Slice(entities, func(i, j int) bool {
		return entities[i].Name < entities[j].Name
	})

	for _, e := range entities {
		s += e.string() + internal.LineFeed()
	}
	return s
}

func (d *Diagram) Build() (err error) {

	if _, writeErr := fmt.Fprint(d.dest, d.String()); writeErr != nil {
		if d.err != nil {
			return fmt.Errorf("failed to write: %w: %s", writeErr, d.err.Error()) //nolint:wrapcheck
		}
		return fmt.Errorf("failed to write: %w", writeErr)
	}
	return d.err
}
