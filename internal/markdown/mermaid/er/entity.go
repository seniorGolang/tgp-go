// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package er

import (
	"fmt"
	"strings"

	"tgp/internal/markdown/internal"
)

type Entity struct {
	// Name is the name of the entity.
	Name string
	// Attributes is the attributes of the entity.
	Attributes []*Attribute
}

func (e *Entity) string() string {
	attrs := make([]string, 0, len(e.Attributes))
	for _, a := range e.Attributes {
		attrs = append(attrs, a.string())
	}

	return fmt.Sprintf(
		"%s%s {%s%s%s%s}",
		"    ", // indent
		e.Name,
		internal.LineFeed(),
		strings.Join(attrs, internal.LineFeed()),
		internal.LineFeed(),
		"    ", // indent
	)
}

func NewEntity(name string, attrs []*Attribute) Entity {
	return Entity{
		Name:       name,
		Attributes: attrs,
	}
}

type Attribute struct {
	// Type is the type of the attribute.
	Type string
	// Name is the name of the attribute.
	Name string
	// IsPrimaryKey is the flag that indicates whether the attribute is a primary key.
	IsPrimaryKey bool
	// IsForeignKey is the flag that indicates whether the attribute is a foreign key.
	IsForeignKey bool
	// IsUniqueKey is the flag that indicates whether the attribute is a unique key.
	IsUniqueKey bool
	// Comment is the comment of the attribute.
	Comment string
}

func (a *Attribute) string() string {
	var keys []string
	if a.IsPrimaryKey {
		keys = append(keys, "PK")
	}
	if a.IsForeignKey {
		keys = append(keys, "FK")
	}
	if a.IsUniqueKey {
		keys = append(keys, "UK")
	}

	s := fmt.Sprintf("        %s %s %s \"%s\"", a.Type, a.Name, strings.Join(keys, ","), a.Comment)
	s = strings.TrimSuffix(s, " ")
	return strings.ReplaceAll(s, "\"\"", "")
}

func (d *Diagram) Relationship(leftE, rightE Entity, leftR, rightR Relationship, identidy Identify, comment string) *Diagram {
	d.body = append(
		d.body,
		fmt.Sprintf("    %s %s%s%s %s : \"%s\"",
			leftE.Name,
			leftR.string(left),
			identidy.string(),
			rightR.string(right),
			rightE.Name,
			comment,
		),
	)

	d.entities.Store(leftE.Name, leftE)
	d.entities.Store(rightE.Name, rightE)

	return d
}

func (d *Diagram) NoRelationship(e Entity) *Diagram {
	d.entities.Store(e.Name, e)
	return d
}
