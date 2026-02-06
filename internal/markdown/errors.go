// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package markdown

import "errors"

var (
	// ErrMismatchColumn is returned when the number of columns in the record doesn't match the header.
	ErrMismatchColumn = errors.New("number of columns in the record doesn't match the header")
	// ErrInitMarkdownIndex is returned when the index can't be initialized.
	ErrInitMarkdownIndex = errors.New("markdown index can't be initialized")
	// ErrCreateMarkdownIndex is returned when the index can't be created.
	ErrCreateMarkdownIndex = errors.New("markdown index can't be created")
	// ErrWriteMarkdownIndex is returned when the index can't be written.
	ErrWriteMarkdownIndex = errors.New("markdown index can't be written")
)
