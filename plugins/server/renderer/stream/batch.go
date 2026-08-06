// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

const maxEnvelopePeek = 16

var (
	ErrBatchTooLarge = errors.New("jsonrpc: batch size exceeded")
	ErrEmptyBatch    = errors.New("jsonrpc: empty batch")
	ErrEmptyBody     = errors.New("jsonrpc: empty body")
	ErrBadEnvelope   = errors.New("jsonrpc: expected object or array")
)

// DecodeEnvelope читает JSON-RPC body: один объект или batch-массив.
// Для массива материализует не больше maxBatch элементов; на следующем — ErrBatchTooLarge.
func DecodeEnvelope[T any](r io.Reader, maxBatch int) (items []T, single bool, err error) {

	var first byte
	if first, err = readFirstNonWhitespace(r); err != nil {
		if errors.Is(err, io.EOF) {
			err = ErrEmptyBody
		}
		return
	}
	if first == 0 {
		err = ErrEmptyBody
		return
	}
	r = io.MultiReader(bytes.NewReader([]byte{first}), r)
	switch first {
	case '{':
		var item T
		if err = json.NewDecoder(r).Decode(&item); err != nil {
			return
		}
		items = []T{item}
		single = true
		if len(items) > maxBatch {
			items = nil
			single = false
			err = ErrBatchTooLarge
		}
		return
	case '[':
		items, err = decodeArray[T](r, maxBatch)
		return
	default:
		err = ErrBadEnvelope
		return
	}
}

func decodeArray[T any](r io.Reader, maxBatch int) (items []T, err error) {

	dec := json.NewDecoder(r)
	var tok json.Token
	if tok, err = dec.Token(); err != nil {
		return
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '[' {
		err = ErrBadEnvelope
		return
	}
	capHint := max(0, maxBatch)
	items = make([]T, 0, capHint)
	for dec.More() {
		if len(items) >= maxBatch {
			items = nil
			err = ErrBatchTooLarge
			return
		}
		var item T
		if err = dec.Decode(&item); err != nil {
			items = nil
			return
		}
		items = append(items, item)
	}
	if tok, err = dec.Token(); err != nil {
		items = nil
		return
	}
	if delim, ok = tok.(json.Delim); !ok || delim != ']' {
		items = nil
		err = ErrBadEnvelope
		return
	}
	if len(items) == 0 {
		items = nil
		err = ErrEmptyBatch
	}
	return
}

func readFirstNonWhitespace(r io.Reader) (firstByte byte, err error) {

	var buf [1]byte
	for range maxEnvelopePeek {
		var n int
		if n, err = r.Read(buf[:]); n == 0 {
			return 0, err
		}
		b := buf[0]
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return b, nil
		}
	}
	return 0, errors.New("leading whitespace exceeds limit")
}
