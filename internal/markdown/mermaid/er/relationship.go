// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package er

type Relationship string

const (
	ZeroToOneRelationship  Relationship = "zero_to_one"
	OneToMoreRelationship  Relationship = "one_to_more"
	ZeroToMoreRelationship Relationship = "zero_to_more"
	ExactlyOneRelationship Relationship = "exactly_one"
)

const (
	left  = true
	right = false
)

func (r Relationship) string(isLeft bool) (s string) {

	switch r {
	case ZeroToOneRelationship:
		if isLeft {
			return "|o"
		}
		return "o|"
	case ExactlyOneRelationship:
		return "||"
	case ZeroToMoreRelationship:
		if isLeft {
			return "}o"
		}
		return "o{"
	case OneToMoreRelationship:
		if isLeft {
			return "}|"
		}
		return "|{"
	default:
		return ""
	}
}
