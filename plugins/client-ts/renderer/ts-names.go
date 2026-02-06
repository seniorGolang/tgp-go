// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import "strings"

var tsReservedToSafe = map[string]string{
	"in":         "input",
	"default":    "defaultValue",
	"class":      "className",
	"type":       "typeName",
	"delete":     "deleteKey",
	"return":     "returnValue",
	"switch":     "switchValue",
	"throw":      "throwValue",
	"try":        "tryValue",
	"var":        "varValue",
	"while":      "whileValue",
	"with":       "withValue",
	"yield":      "yieldValue",
	"let":        "letValue",
	"const":      "constValue",
	"static":     "staticValue",
	"implements": "implementsValue",
	"interface":  "interfaceValue",
	"package":    "packageValue",
	"private":    "privateValue",
	"protected":  "protectedValue",
	"public":     "publicValue",
	"extends":    "extendsValue",
	"enum":       "enumValue",
	"export":     "exportValue",
	"import":     "importValue",
	"await":      "awaitValue",
	"async":      "asyncValue",
	"break":      "breakValue",
	"case":       "caseValue",
	"catch":      "catchValue",
	"continue":   "continueValue",
	"debugger":   "debuggerValue",
	"do":         "doValue",
	"else":       "elseValue",
	"finally":    "finallyValue",
	"for":        "forValue",
	"function":   "functionValue",
	"if":         "ifValue",
	"new":        "newValue",
	"this":       "thisValue",
	"typeof":     "typeofValue",
	"void":       "voidValue",
}

func tsSafeName(name string) string {

	if name == "" {
		return name
	}
	if safe, ok := tsReservedToSafe[strings.ToLower(name)]; ok {
		return safe
	}
	return name
}

func tsLocalVar(name string) string {

	return "_" + name + "_"
}
