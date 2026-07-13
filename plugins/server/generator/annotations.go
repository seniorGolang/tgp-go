// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"

	"tgp/internal/model"
)

func validateServerContract(project *model.Project, contract *model.Contract) (err error) {

	if contract == nil {
		return nil
	}
	if !model.IsAnnotationSet(project, contract, nil, nil, model.TagServerHTTP) {
		return nil
	}

	for _, method := range contract.Methods {
		if err = validateServerMethod(project, contract, method); err != nil {
			return
		}
	}
	return nil
}

func validateServerMethod(project *model.Project, contract *model.Contract, method *model.Method) (err error) {

	if method == nil {
		return nil
	}

	if handlerValue := model.GetAnnotationValue(project, contract, method, nil, model.TagHandler, ""); handlerValue != "" {
		if _, _, err = model.ParseHandlerRef(handlerValue); err != nil {
			return fmt.Errorf("contract %q: method %q: handler: %w", contract.Name, method.Name, err)
		}
	}

	if responseValue := model.GetAnnotationValue(project, contract, method, nil, model.TagHttpResponse, ""); responseValue != "" {
		if _, _, err = model.ParseHandlerRef(responseValue); err != nil {
			return fmt.Errorf("contract %q: method %q: http-response: %w", contract.Name, method.Name, err)
		}
	}
	return nil
}
