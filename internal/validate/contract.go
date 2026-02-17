// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package validate

import (
	"fmt"

	"tgp/internal/model"
)

const tagHttpServer = "http-server"

const (
	typeIDIOReader     = "io:Reader"
	typeIDIOReadCloser = "io:ReadCloser"
)

func Contract(contract *model.Contract, project *model.Project) (err error) {

	if contract == nil {
		return fmt.Errorf("contract cannot be nil")
	}

	for _, method := range contract.Methods {
		for i, arg := range method.Args {
			if arg.Name == "" && arg.TypeID != "context:Context" {
				return fmt.Errorf("contract %q: method %q: argument #%d has no name (all arguments except context.Context must be named)", contract.Name, method.Name, i+1)
			}
		}

		for i, result := range method.Results {
			if result.Name == "" && result.TypeID != "error" {
				return fmt.Errorf("contract %q: method %q: result #%d has no name (all results except error must be named)", contract.Name, method.Name, i+1)
			}
		}

		visited := make(map[string]struct{})

		for _, arg := range method.Args {
			if err = validateVariable(arg, project, contract.Name, method.Name, "argument", visited); err != nil {
				return
			}
		}

		for _, result := range method.Results {
			if err = validateVariable(result, project, contract.Name, method.Name, "result", visited); err != nil {
				return
			}
		}
	}

	if err = validateContractStreamTypes(contract, project); err != nil {
		return
	}

	return
}

func validateContractStreamTypes(contract *model.Contract, project *model.Project) (err error) {

	hasHTTPServer := model.IsAnnotationSet(project, contract, nil, nil, tagHttpServer)

	for _, method := range contract.Methods {
		if !hasHTTPServer {
			for _, arg := range method.Args {
				if arg.TypeID == typeIDIOReader {
					return fmt.Errorf("contract %q: method %q: io.Reader в аргументах разрешён только при аннотации http-server на контракте", contract.Name, method.Name)
				}
			}
			for _, res := range method.Results {
				if res.TypeID == typeIDIOReadCloser {
					return fmt.Errorf("contract %q: method %q: io.ReadCloser в возвращаемых значениях разрешён только при аннотации http-server на контракте", contract.Name, method.Name)
				}
			}
		}
	}

	return
}
