// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package sequence

import "fmt"

type NotePosition string

const (
	// NotePositionOver is a note position.
	NotePositionOver NotePosition = "over"
	// NotePositionRight is a note position.
	NotePositionRight NotePosition = "right of"
	// NotePositionLeft is a note position.
	NotePositionLeft NotePosition = "left of"
)

func (d *Diagram) NoteOver(participant, message string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    note over %s: %s", participant, message))
	return d
}

func (d *Diagram) NoteRightOf(participant, message string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    note right of %s: %s", participant, message))
	return d
}

func (d *Diagram) NoteLeftOf(participant, message string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    note left of %s: %s", participant, message))
	return d
}
