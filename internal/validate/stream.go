// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package validate

import (
	"fmt"
	"strings"

	"tgp/internal/model"
)

func contractStreamAnnotations(project *model.Project, contract *model.Contract) (err error) {

	if contract == nil {
		return nil
	}

	hasWS := model.ContractHasWS(project, contract)
	hasSSE := model.ContractHasSSE(project, contract)

	for _, method := range contract.Methods {
		mode := model.MethodStreamMode(project, contract, method)
		if mode == "" {
			if err = validateNonStreamNoChan(project, contract, method); err != nil {
				return
			}
			continue
		}

		if !hasWS && !hasSSE {
			return fmt.Errorf("contract %q: method %q: stream=%s requires ws-server and/or sse-server on the contract", contract.Name, method.Name, mode)
		}
		if mode != model.StreamModeServer && !hasWS {
			return fmt.Errorf("contract %q: method %q: stream=%s requires ws-server", contract.Name, method.Name, mode)
		}
		if model.IsAnnotationSet(project, contract, method, nil, model.TagSSEPath) && mode != model.StreamModeServer {
			return fmt.Errorf("contract %q: method %q: sse-path is allowed only with stream=server", contract.Name, method.Name)
		}
		if model.IsAnnotationSet(project, contract, method, nil, model.TagHTTPMethod) {
			return fmt.Errorf("contract %q: method %q: stream method cannot have http-method", contract.Name, method.Name)
		}
		if err = validateStreamSignature(project, contract, method, mode); err != nil {
			return
		}
	}

	if hasWS || hasSSE {
		hasStreamMethod := false
		for _, method := range contract.Methods {
			if model.MethodIsStream(project, contract, method) {
				hasStreamMethod = true
				break
			}
		}
		if !hasStreamMethod {
			return fmt.Errorf("contract %q: ws-server/sse-server requires at least one method with stream=server|client|bidi", contract.Name)
		}
	}

	return nil
}

func validateNonStreamNoChan(project *model.Project, contract *model.Contract, method *model.Method) (err error) {

	for _, arg := range method.Args {
		if model.TypeRefIsChan(project, &arg.TypeRef) {
			return fmt.Errorf("contract %q: method %q: argument %q has unsupported type chan (channels are allowed only on stream methods)", contract.Name, method.Name, arg.Name)
		}
	}
	for _, res := range method.Results {
		if model.TypeRefIsChan(project, &res.TypeRef) {
			return fmt.Errorf("contract %q: method %q: result %q has unsupported type chan (channels are allowed only on stream methods)", contract.Name, method.Name, res.Name)
		}
	}
	return nil
}

func validateStreamSignature(project *model.Project, contract *model.Contract, method *model.Method, mode string) (err error) {

	inArg, _, hasIn := model.MethodStreamInChan(project, method)
	outRes, _, hasOut := model.MethodStreamOutChan(project, method)

	chanArgs := 0
	for _, arg := range method.Args {
		if arg.TypeID == "context:Context" {
			continue
		}
		if model.TypeRefIsChan(project, &arg.TypeRef) {
			chanArgs++
		}
	}
	chanResults := 0
	for _, res := range method.Results {
		if res.TypeID == "error" {
			continue
		}
		if model.TypeRefIsChan(project, &res.TypeRef) {
			chanResults++
		}
	}

	switch mode {
	case model.StreamModeServer:
		if !hasOut || chanResults != 1 || chanArgs != 0 {
			return fmt.Errorf("contract %q: method %q: stream=server requires exactly one <-chan result and no chan args", contract.Name, method.Name)
		}
		// Named chan types (ChanOf == nil): direction is checked via Type elsewhere.
		if outRes.ChanOf != nil && outRes.ChanDirection != 0 && outRes.ChanDirection != 2 && outRes.ChanDirection != 3 {
			return fmt.Errorf("contract %q: method %q: stream=server result channel must be receive-only or bidirectional", contract.Name, method.Name)
		}
	case model.StreamModeClient:
		if !hasIn || chanArgs != 1 || chanResults != 0 {
			return fmt.Errorf("contract %q: method %q: stream=client requires exactly one <-chan argument and no chan results", contract.Name, method.Name)
		}
		_ = inArg
	case model.StreamModeBidi:
		if !hasIn || !hasOut || chanArgs != 1 || chanResults != 1 {
			return fmt.Errorf("contract %q: method %q: stream=bidi requires one <-chan argument and one <-chan result", contract.Name, method.Name)
		}
	default:
		return fmt.Errorf("contract %q: method %q: unsupported stream mode %q", contract.Name, method.Name, mode)
	}

	if raw := strings.TrimSpace(model.GetAnnotationValue(project, contract, method, nil, model.TagStream, "")); raw != "" {
		switch strings.ToLower(raw) {
		case model.StreamModeServer, model.StreamModeClient, model.StreamModeBidi:
		default:
			return fmt.Errorf("contract %q: method %q: stream must be server|client|bidi, got %q", contract.Name, method.Name, raw)
		}
	}

	return nil
}
