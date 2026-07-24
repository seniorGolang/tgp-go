// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package validate

import (
	"fmt"
	"strings"

	"tgp/internal/model"
)

var kafkaAcksAllowed = map[string]struct{}{
	model.KafkaAcksAllISR: {},
	model.KafkaAcksLeader: {},
	model.KafkaAcksNoAck:  {},
}

func contractKafkaAnnotations(project *model.Project, contract *model.Contract) (err error) {

	if contract == nil {
		return nil
	}

	if model.ContractHasLegacyKafkaRole(project, contract) {
		return fmt.Errorf("contract %q: @tg kafka-consumer / kafka-publisher are removed; use @tg kafka (see kafka-pub-go / kafka-sub-go)", contract.Name)
	}
	if !model.ContractIsKafka(project, contract) {
		return nil
	}

	if model.ContractIsHTTPFamily(project, contract) {
		return fmt.Errorf("contract %q: kafka contracts cannot combine with http-server/jsonRPC-server/ws-server/sse-server", contract.Name)
	}
	if model.IsAnnotationSet(project, contract, nil, nil, model.TagStream) {
		return fmt.Errorf("contract %q: kafka contracts cannot use stream annotation", contract.Name)
	}

	if len(contract.Methods) == 0 {
		return fmt.Errorf("contract %q: kafka contract requires at least one method", contract.Name)
	}

	if raw := model.ContractKafkaAcks(project, contract); raw != "" {
		if _, ok := kafkaAcksAllowed[raw]; !ok {
			return fmt.Errorf("contract %q: kafka-acks must be noAck, leaderAck or allISRAcks, got %q", contract.Name, raw)
		}
	}

	topics := make(map[string]string)
	for _, method := range contract.Methods {
		if err = validateKafkaMethod(project, contract, method); err != nil {
			return
		}
		topic := model.MethodKafkaTopic(project, contract, method)
		owner := contract.Name + "." + method.Name
		if prev, exists := topics[topic]; exists {
			return fmt.Errorf("contract %q: methods %q and %q share kafka-topic %q (one owner per topic)", contract.Name, prev, method.Name, topic)
		}
		topics[topic] = owner
	}
	return nil
}

// KafkaProject проверяет уникальность топиков среди всех @tg kafka контрактов проекта.
func KafkaProject(project *model.Project) (err error) {

	if project == nil {
		return nil
	}
	owners := make(map[string]string)
	for _, contract := range project.Contracts {
		if !model.ContractIsKafka(project, contract) {
			continue
		}
		for _, method := range contract.Methods {
			topic := model.MethodKafkaTopic(project, contract, method)
			if topic == "" {
				continue
			}
			owner := contract.Name + "." + method.Name
			if prev, exists := owners[topic]; exists {
				return fmt.Errorf("kafka-topic %q is owned by %s and %s (must be unique in contracts-dir)", topic, prev, owner)
			}
			owners[topic] = owner
		}
	}
	return nil
}

func validateKafkaMethod(project *model.Project, contract *model.Contract, method *model.Method) (err error) {

	if model.IsAnnotationSet(project, contract, method, nil, model.TagStream) {
		return fmt.Errorf("contract %q: method %q: stream is not allowed on kafka methods", contract.Name, method.Name)
	}
	if model.IsAnnotationSet(project, contract, method, nil, model.TagHTTPMethod) {
		return fmt.Errorf("contract %q: method %q: http-method is not allowed on kafka methods", contract.Name, method.Name)
	}

	topic := model.MethodKafkaTopic(project, contract, method)
	if topic == "" {
		return fmt.Errorf("contract %q: method %q: kafka-topic is required and must be non-empty after trim", contract.Name, method.Name)
	}

	if methodRaw := strings.TrimSpace(method.Annotations.Value(model.TagKafkaAcks, "")); methodRaw != "" {
		if _, ok := kafkaAcksAllowed[methodRaw]; !ok {
			return fmt.Errorf("contract %q: method %q: kafka-acks must be noAck, leaderAck or allISRAcks, got %q", contract.Name, method.Name, methodRaw)
		}
	}

	for _, arg := range method.Args {
		if model.TypeRefIsChan(project, &arg.TypeRef) {
			return fmt.Errorf("contract %q: method %q: channels are not allowed on kafka methods", contract.Name, method.Name)
		}
	}
	for _, res := range method.Results {
		if model.TypeRefIsChan(project, &res.TypeRef) {
			return fmt.Errorf("contract %q: method %q: channels are not allowed on kafka methods", contract.Name, method.Name)
		}
	}

	if err = validateKafkaResults(contract, method); err != nil {
		return
	}
	if err = validateKafkaContextFirst(contract, method); err != nil {
		return
	}
	return validateKafkaMessageAndBindings(project, contract, method)
}

func validateKafkaResults(contract *model.Contract, method *model.Method) (err error) {

	hasError := false
	for _, res := range method.Results {
		if res.TypeID == "error" {
			hasError = true
			continue
		}
		return fmt.Errorf("contract %q: method %q: kafka methods may only return error", contract.Name, method.Name)
	}
	if !hasError {
		return fmt.Errorf("contract %q: method %q: kafka methods must return error", contract.Name, method.Name)
	}
	return nil
}

func validateKafkaContextFirst(contract *model.Contract, method *model.Method) (err error) {

	if len(method.Args) == 0 || !isContextArgName(method.Args[0]) {
		return fmt.Errorf("contract %q: method %q: first argument must be context.Context", contract.Name, method.Name)
	}
	return nil
}

func validateKafkaMessageAndBindings(project *model.Project, contract *model.Contract, method *model.Method) (err error) {

	explicitName := model.MethodKafkaMessageArgName(project, contract, method)
	message, hasMessage := model.MethodKafkaMessageArg(project, contract, method)
	if explicitName != "" && !hasMessage {
		return fmt.Errorf("contract %q: method %q: kafka-message argument %q not found", contract.Name, method.Name, explicitName)
	}
	if !hasMessage {
		return fmt.Errorf("contract %q: method %q: cannot resolve message argument (set @tg kafka-message or leave exactly one free arg)", contract.Name, method.Name)
	}
	if message.NumberOfPointers != 0 {
		return fmt.Errorf("contract %q: method %q: message argument %q must not be a pointer", contract.Name, method.Name, message.Name)
	}

	keyArg := model.MethodKafkaKeyArg(project, contract, method)
	items := model.MethodKafkaHeaderItems(project, contract, method)
	if raw := strings.TrimSpace(model.GetAnnotationValue(project, contract, method, nil, model.TagKafkaHeaders, "")); raw != "" && len(items) == 0 {
		return fmt.Errorf("contract %q: method %q: invalid kafka-headers format (expected arg|header pairs with non-empty names)", contract.Name, method.Name)
	}

	argByName := make(map[string]*model.Variable, len(method.Args))
	for _, arg := range method.Args {
		argByName[arg.Name] = arg
	}

	if keyArg != "" {
		keyVar, found := argByName[keyArg]
		if !found {
			return fmt.Errorf("contract %q: method %q: kafka-key argument %q not found", contract.Name, method.Name, keyArg)
		}
		if keyArg == message.Name {
			return fmt.Errorf("contract %q: method %q: kafka-key argument %q cannot be the message", contract.Name, method.Name, keyArg)
		}
		if !model.TypeRefIsKafkaKeyOrHeader(&keyVar.TypeRef) {
			return fmt.Errorf("contract %q: method %q: kafka-key argument %q type must be string, []byte, []string or [][]byte", contract.Name, method.Name, keyArg)
		}
	}

	for _, item := range items {
		if strings.TrimSpace(item.Key) == "" {
			return fmt.Errorf("contract %q: method %q: kafka-headers header name must be non-empty after trim", contract.Name, method.Name)
		}
		headerVar, found := argByName[item.Arg]
		if !found {
			return fmt.Errorf("contract %q: method %q: kafka-headers argument %q not found", contract.Name, method.Name, item.Arg)
		}
		if item.Arg == message.Name {
			return fmt.Errorf("contract %q: method %q: kafka-headers argument %q cannot be the message", contract.Name, method.Name, item.Arg)
		}
		if keyArg != "" && item.Arg == keyArg {
			return fmt.Errorf("contract %q: method %q: kafka-headers argument %q cannot be the key", contract.Name, method.Name, item.Arg)
		}
		if !model.TypeRefIsKafkaKeyOrHeader(&headerVar.TypeRef) {
			return fmt.Errorf("contract %q: method %q: kafka-headers argument %q type must be string, []byte, []string or [][]byte", contract.Name, method.Name, item.Arg)
		}
	}

	codec := model.MethodKafkaCodec(project, contract, method)
	if codec == model.KafkaCodecBytes {
		if !model.TypeRefIsByteSlice(&message.TypeRef) && !model.TypeRefIsByteSliceSlice(&message.TypeRef) && !model.TypeRefIsByteSliceEllipsis(&message.TypeRef) {
			return fmt.Errorf("contract %q: method %q: kafka-codec=bytes requires message []byte, [][]byte or ...[]byte", contract.Name, method.Name)
		}
	}
	return nil
}

func isContextArgName(arg *model.Variable) (ok bool) {

	if arg == nil {
		return false
	}
	return arg.TypeID == "context:Context" || arg.TypeID == "context.Context"
}
