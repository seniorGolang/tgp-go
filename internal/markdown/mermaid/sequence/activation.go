// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package sequence

import "fmt"

func (d *Diagram) Activate(participant string) (out *Diagram) {
	d.body = append(d.body, fmt.Sprintf("    activate %s", participant))
	return d
}

func (d *Diagram) Deactivate(participant string) (out *Diagram) {
	d.body = append(d.body, fmt.Sprintf("    deactivate %s", participant))
	return d
}

func (d *Diagram) SyncRequestWithActivation(from, to, message string) (out *Diagram) {
	d.body = append(d.body, fmt.Sprintf("    %s->>+%s: %s", from, to, message))
	return d
}

func (d *Diagram) SyncRequestfWithActivation(from, to, format string, args ...any) (out *Diagram) {
	return d.SyncRequestWithActivation(from, to, fmt.Sprintf(format, args...))
}

func (d *Diagram) SyncResponseWithActivation(from, to, message string) (out *Diagram) {
	d.body = append(d.body, fmt.Sprintf("    %s-->>-%s: %s", from, to, message))
	return d
}

func (d *Diagram) SyncResponsefWithActivation(from, to, format string, args ...any) (out *Diagram) {
	return d.SyncResponseWithActivation(from, to, fmt.Sprintf(format, args...))
}

func (d *Diagram) AsyncRequestWithActivation(from, to, message string) (out *Diagram) {
	d.body = append(d.body, fmt.Sprintf("    %s->>+%s: %s", from, to, message))
	return d
}

func (d *Diagram) AsyncRequestfWithActivation(from, to, format string, args ...any) (out *Diagram) {
	return d.AsyncRequestWithActivation(from, to, fmt.Sprintf(format, args...))
}

func (d *Diagram) AsyncResponseWithActivation(from, to, message string) (out *Diagram) {
	d.body = append(d.body, fmt.Sprintf("    %s-->>-%s: %s", from, to, message))
	return d
}

func (d *Diagram) AsyncResponsefWithActivation(from, to, format string, args ...any) (out *Diagram) {
	return d.AsyncResponseWithActivation(from, to, fmt.Sprintf(format, args...))
}
