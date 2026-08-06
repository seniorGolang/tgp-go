// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package tsg

import "strings"

// ClassDecl пишет заголовок class Name без тела (далее Extends/Implements/Block).
func (s *Statement) ClassDecl(name string) (out *Statement) {

	s.writeIndent()
	s.code.WriteString("class " + name)
	return s
}

// MethodTyped пишет синхронный метод name(params): returnType { ... }.
func (s *Statement) MethodTyped(name string, params *Statement, returnType *Statement, fn func(*Group)) (out *Statement) {

	s.writeIndent()
	s.code.WriteString(name)
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

// While пишет while (condition) { ... }.
func (s *Statement) While(condition *Statement, fn func(*Group)) (out *Statement) {

	s.writeIndent()
	s.code.WriteString("while (")
	if condition != nil {
		s.code.WriteString(condition.String())
	}
	s.code.WriteString(")")
	s.Block(fn)
	return s
}

// TryFinally пишет try { ... } finally { ... }.
func (s *Statement) TryFinally(tryFn func(*Group), finallyFn func(*Group)) (out *Statement) {

	s.writeIndent()
	s.code.WriteString("try {")
	s.code.WriteString("\n")
	s.indent++
	if tryFn != nil {
		tryFn(&Group{statement: s, inObject: false})
	}
	s.indent--
	s.writeIndent()
	s.code.WriteString("} finally {")
	s.code.WriteString("\n")
	s.indent++
	if finallyFn != nil {
		finallyFn(&Group{statement: s, inObject: false})
	}
	s.indent--
	s.writeIndent()
	s.code.WriteString("}")
	return s
}

// TryCatch пишет try/catch; пустой catchParam даёт catch { без биндинга.
func (s *Statement) TryCatch(tryFn func(*Group), catchFn func(*Group), catchParam string) (out *Statement) {

	s.writeIndent()
	s.code.WriteString("try {")
	s.code.WriteString("\n")
	s.indent++
	if tryFn != nil {
		tryFn(&Group{statement: s, inObject: false})
	}
	s.indent--
	s.writeIndent()
	if catchParam == "" {
		s.code.WriteString("} catch {")
	} else {
		s.code.WriteString("} catch (" + catchParam + ") {")
	}
	s.code.WriteString("\n")
	s.indent++
	if catchFn != nil {
		catchFn(&Group{statement: s, inObject: false})
	}
	s.indent--
	s.writeIndent()
	s.code.WriteString("}")
	return s
}

// Continue пишет continue;.
func (s *Statement) Continue() (out *Statement) {

	s.writeIndent()
	s.code.WriteString("continue;")
	return s
}

// VoidExpr пишет void expr;.
func (s *Statement) VoidExpr(expr *Statement) (out *Statement) {

	s.writeIndent()
	s.code.WriteString("void ")
	if expr != nil {
		s.code.WriteString(expr.String())
	}
	s.code.WriteString(";")
	return s
}

// Bang пишет ! (non-null assertion / negation prefix без пробела).
func (s *Statement) Bang() (out *Statement) {

	s.code.WriteString("!")
	return s
}

// Paren оборачивает выражение в скобки.
func (s *Statement) Paren(expr *Statement) (out *Statement) {

	s.code.WriteString("(")
	if expr != nil {
		s.code.WriteString(expr.String())
	}
	s.code.WriteString(")")
	return s
}

// InExpr пишет prop in obj без кавычек вокруг prop.
func (s *Statement) InExpr(prop *Statement, obj *Statement) (out *Statement) {

	if prop != nil {
		s.code.WriteString(prop.String())
	}
	s.code.WriteString(" in ")
	if obj != nil {
		s.code.WriteString(obj.String())
	}
	return s
}

// While пишет while на уровне тела.
func (g *Group) While(condition *Statement, fn func(*Group)) {

	g.statement.writeIndent()
	g.statement.code.WriteString("while (")
	if condition != nil {
		g.statement.code.WriteString(condition.String())
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

// TryFinally пишет try/finally на уровне тела.
func (g *Group) TryFinally(tryFn func(*Group), finallyFn func(*Group)) {

	g.statement.writeIndent()
	g.statement.code.WriteString("try {")
	g.statement.code.WriteString("\n")
	g.statement.indent++
	if tryFn != nil {
		tryFn(g)
	}
	g.statement.indent--
	g.statement.writeIndent()
	g.statement.code.WriteString("} finally {")
	g.statement.code.WriteString("\n")
	g.statement.indent++
	if finallyFn != nil {
		finallyFn(g)
	}
	g.statement.indent--
	g.statement.writeIndent()
	g.statement.code.WriteString("}\n")
}

// TryCatch пишет try/catch; пустой catchParam даёт catch { без биндинга.
func (g *Group) TryCatch(tryFn func(*Group), catchFn func(*Group), catchParam string) {

	g.statement.writeIndent()
	g.statement.code.WriteString("try {")
	g.statement.code.WriteString("\n")
	g.statement.indent++
	if tryFn != nil {
		tryFn(g)
	}
	g.statement.indent--
	g.statement.writeIndent()
	if catchParam == "" {
		g.statement.code.WriteString("} catch {")
	} else {
		g.statement.code.WriteString("} catch (" + catchParam + ") {")
	}
	g.statement.code.WriteString("\n")
	g.statement.indent++
	if catchFn != nil {
		catchFn(g)
	}
	g.statement.indent--
	g.statement.writeIndent()
	g.statement.code.WriteString("}\n")
}

// Continue пишет continue; на уровне тела.
func (g *Group) Continue() {

	g.statement.writeIndent()
	g.statement.code.WriteString("continue;\n")
}

// ElseIf пишет else if после If (съедает хвостовой \n у предыдущего блока).
func (g *Group) ElseIf(condition *Statement, fn func(*Group)) {

	str := g.statement.code.String()
	if strings.HasSuffix(str, "\n") {
		g.statement.code.Reset()
		g.statement.code.WriteString(strings.TrimSuffix(str, "\n"))
	}
	g.statement.code.WriteString(" else if (")
	if condition != nil {
		g.statement.code.WriteString(condition.String())
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

// Else пишет else после If/ElseIf.
func (g *Group) Else(fn func(*Group)) {

	str := g.statement.code.String()
	if strings.HasSuffix(str, "\n") {
		g.statement.code.Reset()
		g.statement.code.WriteString(strings.TrimSuffix(str, "\n"))
	}
	g.statement.code.WriteString(" else {")
	g.statement.code.WriteString("\n")
	g.statement.indent++
	if fn != nil {
		fn(g)
	}
	g.statement.indent--
	g.statement.writeIndent()
	g.statement.code.WriteString("}\n")
}
