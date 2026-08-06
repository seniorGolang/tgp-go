// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

// JSONRPCWireMethod — каноническое имя JSON-RPC method в wire: lowerCamel(contract).lowerCamel(method).
func JSONRPCWireMethod(contractName, methodName string) (wireMethod string) {

	return LowerCamel(contractName) + "." + LowerCamel(methodName)
}

// LowerCamel преобразует имя из контракта (PascalCase) в camelCase для wire и HTTP path.
func LowerCamel(name string) (result string) {

	return lowerCamel(name)
}
