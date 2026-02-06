// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package sequence

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

	return &Diagram{
		body:   []string{"sequenceDiagram"},
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

func (d *Diagram) SyncRequest(from, to, message string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    %s->>%s: %s", from, to, message))
	return d
}

func (d *Diagram) SyncRequestf(from, to, format string, args ...any) *Diagram {
	return d.SyncRequest(from, to, fmt.Sprintf(format, args...))
}

func (d *Diagram) SyncResponse(from, to, message string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    %s-->>%s: %s", from, to, message))
	return d
}

func (d *Diagram) SyncResponsef(from, to, format string, args ...any) *Diagram {
	return d.SyncResponse(from, to, fmt.Sprintf(format, args...))
}

func (d *Diagram) RequestError(from, to, message string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    %s-x%s: %s", from, to, message))
	return d
}

func (d *Diagram) RequestErrorf(from, to, format string, args ...any) *Diagram {
	return d.RequestError(from, to, fmt.Sprintf(format, args...))
}

func (d *Diagram) ResponseError(from, to, message string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    %s--x%s: %s", from, to, message))
	return d
}

func (d *Diagram) ResponseErrorf(from, to, format string, args ...any) *Diagram {
	return d.ResponseError(from, to, fmt.Sprintf(format, args...))
}

func (d *Diagram) AsyncRequest(from, to, message string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    %s->)%s: %s", from, to, message))
	return d
}

func (d *Diagram) AsyncRequestf(from, to, format string, args ...any) *Diagram {
	return d.AsyncRequest(from, to, fmt.Sprintf(format, args...))
}

func (d *Diagram) AsyncResponse(from, to, message string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    %s--)%s: %s", from, to, message))
	return d
}

func (d *Diagram) AsyncResponsef(from, to, format string, args ...any) *Diagram {
	return d.AsyncResponse(from, to, fmt.Sprintf(format, args...))
}

func (d *Diagram) LF() *Diagram {
	d.body = append(d.body, "")
	return d
}
