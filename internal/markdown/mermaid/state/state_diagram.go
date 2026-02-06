// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package state

import (
	"fmt"
	"io"
	"strings"

	"tgp/internal/markdown/internal"
)

type Diagram struct {
	body   []string
	config *config
	dest   io.Writer
	err    error
}

func NewDiagram(w io.Writer, opts ...Option) *Diagram {
	c := newConfig()

	for _, opt := range opts {
		opt(c)
	}

	lines := []string{}
	if c.title != noTitle {
		lines = append(lines, "---")
		lines = append(lines, fmt.Sprintf("title: %s", c.title))
		lines = append(lines, "---")
	}
	lines = append(lines, "stateDiagram-v2")

	return &Diagram{
		body:   lines,
		dest:   w,
		config: c,
	}
}

func (d *Diagram) String() string {
	return strings.Join(d.body, internal.LineFeed())
}

func (d *Diagram) Error() error {
	return d.err
}

func (d *Diagram) Build() error {
	if _, err := fmt.Fprint(d.dest, d.String()); err != nil {
		if d.err != nil {
			return fmt.Errorf("failed to write: %w: %s", err, d.err.Error()) //nolint:wrapcheck
		}
		return fmt.Errorf("failed to write: %w", err)
	}
	return nil
}

func (d *Diagram) State(id, description string) *Diagram {
	if description == "" {
		d.body = append(d.body, fmt.Sprintf("    %s", id))
	} else {
		d.body = append(d.body, fmt.Sprintf("    %s : %s", id, description))
	}
	return d
}

func (d *Diagram) Transition(from, to string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    %s --> %s", from, to))
	return d
}

func (d *Diagram) TransitionWithNote(from, to, note string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    %s --> %s : %s", from, to, note))
	return d
}

func (d *Diagram) StartTransition(to string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    [*] --> %s", to))
	return d
}

func (d *Diagram) StartTransitionWithNote(to, note string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    [*] --> %s : %s", to, note))
	return d
}

func (d *Diagram) EndTransition(from string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    %s --> [*]", from))
	return d
}

func (d *Diagram) EndTransitionWithNote(from, note string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    %s --> [*] : %s", from, note))
	return d
}

func (d *Diagram) NoteLeft(state, note string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    note left of %s : %s", state, note))
	return d
}

func (d *Diagram) NoteRight(state, note string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    note right of %s : %s", state, note))
	return d
}

func (d *Diagram) NoteLeftMultiLine(state string, lines ...string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    note left of %s", state))
	for _, line := range lines {
		d.body = append(d.body, fmt.Sprintf("        %s", line))
	}
	d.body = append(d.body, "    end note")
	return d
}

func (d *Diagram) NoteRightMultiLine(state string, lines ...string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    note right of %s", state))
	for _, line := range lines {
		d.body = append(d.body, fmt.Sprintf("        %s", line))
	}
	d.body = append(d.body, "    end note")
	return d
}

func (d *Diagram) CompositeState(id string) *CompositeStateBuilder {
	return &CompositeStateBuilder{
		diagram: d,
		id:      id,
	}
}

type CompositeStateBuilder struct {
	diagram *Diagram
	id      string
	body    []string
}

func (b *CompositeStateBuilder) State(id, description string) *CompositeStateBuilder {
	if description == "" {
		b.body = append(b.body, fmt.Sprintf("        %s", id))
	} else {
		b.body = append(b.body, fmt.Sprintf("        %s : %s", id, description))
	}
	return b
}

func (b *CompositeStateBuilder) Transition(from, to string) *CompositeStateBuilder {
	b.body = append(b.body, fmt.Sprintf("        %s --> %s", from, to))
	return b
}

func (b *CompositeStateBuilder) TransitionWithNote(from, to, note string) *CompositeStateBuilder {
	b.body = append(b.body, fmt.Sprintf("        %s --> %s : %s", from, to, note))
	return b
}

func (b *CompositeStateBuilder) StartTransition(to string) *CompositeStateBuilder {
	b.body = append(b.body, fmt.Sprintf("        [*] --> %s", to))
	return b
}

func (b *CompositeStateBuilder) EndTransition(from string) *CompositeStateBuilder {
	b.body = append(b.body, fmt.Sprintf("        %s --> [*]", from))
	return b
}

func (b *CompositeStateBuilder) End() *Diagram {
	b.diagram.body = append(b.diagram.body, fmt.Sprintf("    state %s {", b.id))
	b.diagram.body = append(b.diagram.body, b.body...)
	b.diagram.body = append(b.diagram.body, "    }")
	return b.diagram
}

func (d *Diagram) Fork(id string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    state %s <<fork>>", id))
	return d
}

func (d *Diagram) Join(id string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    state %s <<join>>", id))
	return d
}

func (d *Diagram) Choice(id string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    state %s <<choice>>", id))
	return d
}

func (d *Diagram) LF() *Diagram {
	d.body = append(d.body, "")
	return d
}

type Direction string

const (
	DirectionLR Direction = "LR"
	DirectionRL Direction = "RL"
	DirectionTB Direction = "TB"
	DirectionBT Direction = "BT"
)

func (d *Diagram) SetDirection(dir Direction) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    direction %s", dir))
	return d
}

func (d *Diagram) Concurrent() *Diagram {
	d.body = append(d.body, "    ---")
	return d
}
