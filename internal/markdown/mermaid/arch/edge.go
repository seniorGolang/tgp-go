// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package arch

import "fmt"

type Position string

const (
	// PositionLeft is the left position.
	PositionLeft Position = "L"
	// PositionRight is the right position.
	PositionRight Position = "R"
	// PositionTop is the top position.
	PositionTop Position = "T"
	// PositionBottom is the bottom position.
	PositionBottom Position = "B"
)

type Arrow string

const (
	// ArrowNone is the default arrow.
	ArrowNone Arrow = ""
	// ArrowRight is the right arrow.
	ArrowRight Arrow = ">"
	// ArrowLeft is the left arrow.
	ArrowLeft Arrow = "<"
)

type Edge struct {
	// ServiceID is edge's service ID.
	// A junction ID can be specified instead of a service ID.
	ServiceID string
	// Position is edge's position. Top, Bottom, Left, Right.
	Position Position
	// Arrow is edge's arrow. None, Left, Right.
	Arrow Arrow
}

func (a *Architecture) Edges(from, to Edge) *Architecture {
	a.body = append(
		a.body,
		fmt.Sprintf("    %s:%s %s--%s %s:%s",
			from.ServiceID, from.Position, from.Arrow,
			to.Arrow, to.Position, to.ServiceID,
		),
	)
	return a
}

func (a *Architecture) EdgesInAnothorGroup(from, to Edge) *Architecture {
	a.body = append(
		a.body,
		fmt.Sprintf("    %s{group}:%s %s--%s %s:%s{group}",
			from.ServiceID, from.Position, from.Arrow,
			to.Arrow, to.Position, to.ServiceID,
		),
	)
	return a
}
