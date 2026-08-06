// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package sequence

import (
	"strings"
)

func (d *Diagram) AutoNumber() (out *Diagram) {
	d.body = append(d.body, "    autonumber")
	return d
}

func (d *Diagram) BoxStart(participant []string) (out *Diagram) {
	d.body = append(d.body, "    box "+strings.Join(participant, " & "))
	return d
}

func (d *Diagram) BoxEnd() (out *Diagram) {
	d.body = append(d.body, "    end")
	return d
}

func (d *Diagram) Participant(participant string) (out *Diagram) {
	d.body = append(d.body, "    participant "+participant)
	return d
}

func (d *Diagram) Actor(actor string) (out *Diagram) {
	d.body = append(d.body, "    actor "+actor)
	return d
}

func (d *Diagram) CreateParticipant(participant string) (out *Diagram) {
	d.body = append(d.body, "    create participant "+participant)
	return d
}

func (d *Diagram) DestroyParticipant(participant string) (out *Diagram) {
	d.body = append(d.body, "    destroy "+participant)
	return d
}

func (d *Diagram) CreateActor(actor string) (out *Diagram) {
	d.body = append(d.body, "    create actor "+actor)
	return d
}

func (d *Diagram) DestroyActor(actor string) (out *Diagram) {
	d.body = append(d.body, "    destroy "+actor)
	return d
}
