// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package types

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

type Generator struct {
	project   *model.Project
	srcFile   SrcFile
	typeCache map[string]*Statement
}

func NewGenerator(project *model.Project, srcFile SrcFile) *Generator {
	return &Generator{
		project:   project,
		srcFile:   srcFile,
		typeCache: make(map[string]*Statement),
	}
}

func (g *Generator) FieldTypeFromVariable(variable *model.Variable, allowEllipsis bool) *Statement {
	return g.FieldTypeFromTypeRef(&variable.TypeRef, allowEllipsis)
}

func (g *Generator) FieldTypeFromTypeRef(typeRef *model.TypeRef, allowEllipsis bool) *Statement {
	c := &Statement{}

	if typeRef.IsEllipsis && allowEllipsis {
		c.Op("...")
		if typeRef.TypeID != "" {
			return c.Add(g.FieldType(typeRef.TypeID, typeRef.NumberOfPointers, false))
		}
		if typeRef.MapKey != nil && typeRef.MapValue != nil {
			keyType := g.FieldTypeFromTypeRef(typeRef.MapKey, false)
			valueType := g.FieldTypeFromTypeRef(typeRef.MapValue, false)
			return c.Map(keyType).Add(valueType)
		}
		return c
	}

	if typeRef.IsSlice || typeRef.ArrayLen > 0 {
		for i := 0; i < typeRef.NumberOfPointers; i++ {
			c.Op("*")
		}
		if typeRef.IsSlice {
			c.Index()
		} else {
			c.Index(Lit(typeRef.ArrayLen))
		}
		if typeRef.TypeID != "" {
			return c.Add(g.FieldType(typeRef.TypeID, typeRef.ElementPointers, false))
		}
		if typeRef.MapKey != nil && typeRef.MapValue != nil {
			keyType := g.FieldTypeFromTypeRef(typeRef.MapKey, false)
			valueType := g.FieldTypeFromTypeRef(typeRef.MapValue, false)
			return c.Map(keyType).Add(valueType)
		}
		return c
	}

	if typeRef.MapKey != nil && typeRef.MapValue != nil {
		for i := 0; i < typeRef.NumberOfPointers; i++ {
			c.Op("*")
		}
		keyType := g.FieldTypeFromTypeRef(typeRef.MapKey, false)
		valueType := g.FieldTypeFromTypeRef(typeRef.MapValue, false)
		return c.Map(keyType).Add(valueType)
	}

	return c.Add(g.FieldType(typeRef.TypeID, typeRef.NumberOfPointers, false))
}

func (g *Generator) FieldType(typeID string, numberOfPointers int, allowEllipsis bool) *Statement {

	cacheKey := fmt.Sprintf("%s:%d:%v", typeID, numberOfPointers, allowEllipsis)
	if cached, ok := g.typeCache[cacheKey]; ok {
		if cacheLogger != nil {
			cacheLogger.OnCacheHit()
		}
		return cached
	}

	// Логируем промах кэша
	if cacheLogger != nil {
		cacheLogger.OnCacheMiss()
	}

	result := g.fieldTypeImpl(typeID, numberOfPointers, allowEllipsis)

	// Сохраняем в кэш
	g.typeCache[cacheKey] = result
	return result
}

func (g *Generator) fieldTypeImpl(typeID string, numberOfPointers int, allowEllipsis bool) *Statement {

	c := &Statement{}

	for i := 0; i < numberOfPointers; i++ {
		c.Op("*")
	}

	typ, ok := g.project.Types[typeID]
	if !ok {
		return g.fieldTypeFromID(typeID, c)
	}

	if typ.IsEllipsis && allowEllipsis {
		c.Op("...")
		if typ.ArrayOfID != "" {
			return c.Add(g.FieldType(typ.ArrayOfID, 0, false))
		}
		return c
	}

	var result *Statement
	switch typ.Kind {
	case model.TypeKindArray:
		result = g.fieldTypeArray(typ, c)
	case model.TypeKindMap:
		result = g.fieldTypeMap(typ, c)
	case model.TypeKindChan:
		result = g.fieldTypeChan(typ, c)
	case model.TypeKindStruct:
		result = g.fieldTypeStruct(typ, c)
	case model.TypeKindInterface:
		result = g.fieldTypeInterface(typ, c)
	case model.TypeKindFunction:
		result = g.fieldTypeFunction(typ, c)
	case model.TypeKindAlias:
		result = g.fieldTypeAlias(typ, numberOfPointers, allowEllipsis, c)
	default:
		result = g.fieldTypePrimitive(typ, typeID, c)
	}

	return result
}

func (g *Generator) fieldTypeFromID(typeID string, c *Statement) *Statement {
	if strings.Contains(typeID, ":") {
		parts := strings.SplitN(typeID, ":", 2)
		if len(parts) == 2 && parts[1] != "" {
			pkgPath := parts[0]
			typeName := parts[1]
			g.srcFile.ImportName(pkgPath, filepath.Base(pkgPath))
			return c.Qual(pkgPath, typeName)
		}
	}
	return c.Id(typeID)
}

func (g *Generator) fieldTypeArray(typ *model.Type, c *Statement) *Statement {
	// Если это именованный тип, который определен как массив
	if typ.TypeName != "" && (typ.ImportPkgPath != "" || typ.PkgName != "") {
		return g.fieldTypeNamed(typ, c)
	}

	// Обычный массив/слайс
	switch {
	case typ.IsSlice:
		c.Index()
	case typ.ArrayLen > 0:
		c.Index(Lit(typ.ArrayLen))
	default:
		c.Index()
	}
	if typ.ArrayOfID != "" {
		return c.Add(g.FieldType(typ.ArrayOfID, 0, false))
	}
	return c
}

func (g *Generator) fieldTypeMap(typ *model.Type, c *Statement) *Statement {
	if typ.TypeName != "" && (typ.ImportPkgPath != "" || typ.PkgName != "") {
		return g.fieldTypeNamed(typ, c)
	}

	if typ.MapKey != nil && typ.MapValue != nil {
		keyType := g.FieldTypeFromTypeRef(typ.MapKey, false)
		valueType := g.FieldTypeFromTypeRef(typ.MapValue, false)
		return c.Map(keyType).Add(valueType)
	}
	return c
}

func (g *Generator) fieldTypeChan(typ *model.Type, c *Statement) *Statement {
	switch typ.ChanDirection {
	case 1: // send only
		c.Chan().Op("<-")
	case 2: // receive only
		c.Op("<-").Chan()
	default: // both
		c.Chan()
	}
	if typ.ChanOfID != "" {
		return c.Add(g.FieldType(typ.ChanOfID, 0, false))
	}
	return c
}

func (g *Generator) fieldTypeStruct(typ *model.Type, c *Statement) *Statement {
	return g.fieldTypeNamed(typ, c)
}

func (g *Generator) fieldTypeInterface(typ *model.Type, c *Statement) *Statement {
	return g.fieldTypeNamed(typ, c)
}

func (g *Generator) fieldTypeFunction(typ *model.Type, c *Statement) *Statement {
	args := make([]Code, 0, len(typ.FunctionArgs))
	for _, arg := range typ.FunctionArgs {
		argType := g.FieldTypeFromVariable(arg, false)
		args = append(args, Id(toLowerCamel(arg.Name)).Add(argType))
	}
	results := make([]Code, 0, len(typ.FunctionResults))
	for _, res := range typ.FunctionResults {
		resType := g.FieldTypeFromVariable(res, false)
		results = append(results, resType)
	}
	return c.Func().Params(args...).Params(results...)
}

func (g *Generator) fieldTypeAlias(typ *model.Type, numberOfPointers int, allowEllipsis bool, c *Statement) *Statement {
	if typ.AliasOf != "" {
		return g.FieldType(typ.AliasOf, numberOfPointers, allowEllipsis)
	}
	return c.Id(typ.TypeName)
}

func (g *Generator) fieldTypePrimitive(typ *model.Type, typeID string, c *Statement) *Statement {
	// Если у типа есть ImportPkgPath и TypeName, это именованный тип из другого пакета
	if typ.ImportPkgPath != "" && typ.TypeName != "" {
		return g.fieldTypeNamed(typ, c)
	}

	// Встроенный базовый тип
	if typ.Kind != "" {
		return c.Id(string(typ.Kind))
	}
	return c.Id(typeID)
}

func (g *Generator) fieldTypeNamed(typ *model.Type, c *Statement) *Statement {
	if typ.ImportPkgPath != "" {
		packageName := typ.PkgName
		if packageName == "" {
			packageName = filepath.Base(typ.ImportPkgPath)
		}
		if typ.ImportAlias != "" && typ.ImportAlias != packageName {
			g.srcFile.ImportAlias(typ.ImportPkgPath, typ.ImportAlias)
			return c.Qual(typ.ImportPkgPath, typ.TypeName)
		}
		g.srcFile.ImportName(typ.ImportPkgPath, packageName)
		return c.Qual(typ.ImportPkgPath, typ.TypeName)
	}
	return c.Id(typ.TypeName)
}

func (g *Generator) FuncDefinitionParams(vars []*model.Variable) *Statement {
	c := &Statement{}
	c.ListFunc(func(gr *Group) {
		for _, v := range vars {
			typeCode := g.FieldTypeFromVariable(v, true)
			gr.Id(toLowerCamel(v.Name)).Add(typeCode)
		}
	})
	return c
}

func toLowerCamel(s string) string {

	if s == "" {
		return s
	}
	allUpper := true
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			allUpper = false
			break
		}
	}
	if allUpper {
		return s
	}
	// Если первый символ в верхнем регистре, делаем его нижним
	runes := []rune(s)
	if len(runes) > 0 && runes[0] >= 'A' && runes[0] <= 'Z' {
		runes[0] = unicode.ToLower(runes[0])
		s = string(runes)
	}
	// Применяем toCamel, но с capNext = false
	s = strings.Trim(s, " ")
	result := strings.Builder{}
	capNext := false
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r)
			capNext = false
		case r >= '0' && r <= '9':
			result.WriteRune(r)
			capNext = false
		case r >= 'a' && r <= 'z':
			if capNext {
				result.WriteRune(unicode.ToUpper(r))
			} else {
				result.WriteRune(r)
			}
			capNext = false
		case r == '_' || r == ' ' || r == '-' || r == '.':
			capNext = true
		default:
			result.WriteRune(r)
			capNext = false
		}
	}
	return result.String()
}
