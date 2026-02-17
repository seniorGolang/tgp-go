// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package sequence

import (
	"fmt"
	"strings"
)

func (d *Diagram) AutoNumber() (out *Diagram) {
	d.body = append(d.body, "    autonumber")
	return d
}

func (d *Diagram) BoxStart(participant []string) (out *Diagram) {
	d.body = append(d.body, fmt.Sprintf("    box %s", strings.Join(participant, " & ")))
	return d
}

func (d *Diagram) BoxEnd() (out *Diagram) {
	d.body = append(d.body, "    end")
	return d
}

func (d *Diagram) Participant(participant string) (out *Diagram) {
	d.body = append(d.body, fmt.Sprintf("    participant %s", participant))
	return d
}

func (d *Diagram) Actor(actor string) (out *Diagram) {
	d.body = append(d.body, fmt.Sprintf("    actor %s", actor))
	return d
}

func (d *Diagram) CreateParticipant(participant string) (out *Diagram) {
	d.body = append(d.body, fmt.Sprintf("    create participant %s", participant))
	return d
}

func (d *Diagram) DestroyParticipant(participant string) (out *Diagram) {
	d.body = append(d.body, fmt.Sprintf("    destroy %s", participant))
	return d
}

func (d *Diagram) CreateActor(actor string) (out *Diagram) {
	d.body = append(d.body, fmt.Sprintf("    create actor %s", actor))
	return d
}

func (d *Diagram) DestroyActor(actor string) (out *Diagram) {
	d.body = append(d.body, fmt.Sprintf("    destroy %s", actor))
	return d
}
