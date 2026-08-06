// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package tsg

// AsyncMethod пишет async-метод с готовым returnType (без авто-обёртки Promise).
func (s *Statement) AsyncMethod(name string, params *Statement, returnType *Statement, fn func(*Group)) (out *Statement) {

	return s.writeAsyncMethod("async "+name, "", params, returnType, fn)
}

// AsyncMethodGeneric пишет async name<TypeParams>(...): returnType { ... }.
func (s *Statement) AsyncMethodGeneric(name string, typeParams string, params *Statement, returnType *Statement, fn func(*Group)) (out *Statement) {

	return s.writeAsyncMethod("async "+name, typeParams, params, returnType, fn)
}

// AsyncGeneratorMethod пишет async-generator метод: async *name(...): returnType { ... }.
func (s *Statement) AsyncGeneratorMethod(name string, params *Statement, returnType *Statement, fn func(*Group)) (out *Statement) {

	return s.writeAsyncMethod("async *"+name, "", params, returnType, fn)
}

// AsyncGeneratorMethodGeneric пишет async *name<TypeParams>(...): returnType { ... }.
func (s *Statement) AsyncGeneratorMethodGeneric(name string, typeParams string, params *Statement, returnType *Statement, fn func(*Group)) (out *Statement) {

	return s.writeAsyncMethod("async *"+name, typeParams, params, returnType, fn)
}

func (s *Statement) writeAsyncMethod(prefix string, typeParams string, params *Statement, returnType *Statement, fn func(*Group)) (out *Statement) {

	s.writeIndent()
	s.code.WriteString(prefix)
	if typeParams != "" {
		s.code.WriteString("<" + typeParams + ">")
	}
	if params != nil {
		s.code.WriteString(params.String())
	} else {
		s.code.WriteString("()")
	}
	if returnType != nil {
		s.code.WriteString(": ")
		s.code.WriteString(returnType.String())
	}
	s.code.WriteString(" {")
	s.code.WriteString("\n")
	s.indent++
	if fn != nil {
		fn(&Group{statement: s, inObject: false})
	}
	s.indent--
	s.writeIndent()
	s.code.WriteString("}")
	return s
}

// ForAwait пишет for await (const variable of iterable) { ... }.
func (s *Statement) ForAwait(variable string, iterable *Statement, fn func(*Group)) (out *Statement) {

	s.writeIndent()
	s.code.WriteString("for await (const " + variable + " of ")
	if iterable != nil {
		s.code.WriteString(iterable.String())
	}
	s.code.WriteString(")")
	s.Block(fn)
	return s
}

// Yield пишет yield expr;.
func (s *Statement) Yield(expr *Statement) (out *Statement) {

	s.writeIndent()
	s.code.WriteString("yield ")
	if expr != nil {
		s.code.WriteString(expr.String())
	}
	s.code.WriteString(";")
	return s
}

// ForAwait пишет for await на уровне тела метода/блока.
func (g *Group) ForAwait(variable string, iterable *Statement, fn func(*Group)) {

	g.statement.writeIndent()
	g.statement.code.WriteString("for await (const " + variable + " of ")
	if iterable != nil {
		g.statement.code.WriteString(iterable.String())
	}
	g.statement.code.WriteString(") {")
	g.statement.code.WriteString("\n")
	g.statement.indent++
	if fn != nil {
		fn(g)
	}
	g.statement.indent--
	g.statement.writeIndent()
	g.statement.code.WriteString("}\n")
}

// Yield пишет yield expr; на уровне тела метода/блока.
func (g *Group) Yield(expr *Statement) {

	g.statement.writeIndent()
	g.statement.code.WriteString("yield ")
	if expr != nil {
		g.statement.code.WriteString(expr.String())
	}
	g.statement.code.WriteString(";\n")
}
