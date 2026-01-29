// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package er

type Identify bool

const (
	// Identifying is a constant that represents an identifying relationship.
	// It represents "--" in the entity relationship diagram.
	Identifying Identify = true
	// NonIdentifying is a constant that represents a non-identifying relationship.
	// It represents ".." in the entity relationship diagram.
	NonIdentifying Identify = false
)

func (i Identify) string() string {
	if i == Identifying {
		return "--"
	}
	return ".."
}
