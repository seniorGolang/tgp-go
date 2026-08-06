// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
)

type gate struct {
	reader *multipart.Reader
	names  []string
	types  []string
	index  int
	part   *multipart.Part
}

type partReader struct {
	gate  *gate
	index int
}

// Parts is an ordered multipart session over a single reader.
// Readers must be consumed in name order; Close drains the active part.
type Parts struct {
	gate    *gate
	readers []*partReader
}

// NewParts returns an ordered multipart session.
// Parts must be read in the order of names; out-of-order Read returns an error.
func NewParts(reader *multipart.Reader, names []string, contentTypes []string) (parts *Parts) {

	shared := &gate{reader: reader, names: names, types: contentTypes}
	parts = &Parts{gate: shared, readers: make([]*partReader, len(names))}
	for i := range names {
		parts.readers[i] = &partReader{gate: shared, index: i}
	}
	return parts
}

// Reader returns the io.Reader for the part at index.
func (p *Parts) Reader(index int) (r io.Reader) {

	return p.readers[index]
}

// Close drains and closes the currently open part, if any. Safe to call multiple times.
func (p *Parts) Close() (err error) {

	if p == nil {
		return nil
	}
	return p.gate.closeActive()
}

func (g *gate) closeActive() (err error) {

	if g.part == nil {
		return nil
	}
	err = g.part.Close()
	g.part = nil
	g.index++
	return err
}

func (p *partReader) Read(buf []byte) (n int, err error) {

	if p.index < p.gate.index {
		return 0, io.EOF
	}
	if p.index > p.gate.index {
		return 0, fmt.Errorf("multipart part %q: read out of order, expected %q", p.gate.names[p.index], p.gate.names[p.gate.index])
	}
	if p.gate.part == nil {
		if openErr := p.gate.open(); openErr != nil {
			return 0, openErr
		}
	}
	n, err = p.gate.part.Read(buf)
	if errors.Is(err, io.EOF) {
		_ = p.gate.part.Close()
		p.gate.part = nil
		p.gate.index++
	}
	return n, err
}

func (g *gate) open() (err error) {

	want := g.names[g.index]
	for {
		var part *multipart.Part
		if part, err = g.reader.NextPart(); err != nil {
			if err == io.EOF {
				return fmt.Errorf("multipart part %q: not found", want)
			}
			return err
		}
		if part.FormName() != want {
			_ = part.Close()
			continue
		}
		if len(g.types) > g.index && g.types[g.index] != "" {
			if got := part.Header.Get("Content-Type"); got != g.types[g.index] {
				_ = part.Close()
				return fmt.Errorf("multipart part %q: invalid content-type %q", want, got)
			}
		}
		g.part = part
		return nil
	}
}
