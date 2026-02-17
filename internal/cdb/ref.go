package cdb

import (
	"fmt"
	"strings"
)

// Ref — распарсенный идентификатор контракта: projectKey[:contracts][@version].
type Ref struct {
	ProjectKey string
	Contracts  []string
	Version    string
}

// ParseRef разбирает строку ref в Ref. Поддерживаются форматы:
// project:contracts@version и project@version:contracts. Алиасы не раскрываются (это делает ResolveAlias).
func ParseRef(ref string) (parsed Ref, err error) {

	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Ref{}, fmt.Errorf("empty ref")
	}

	at := strings.LastIndex(ref, "@")
	var head, tail string
	if at >= 0 {
		head = strings.TrimSpace(ref[:at])
		tail = strings.TrimSpace(ref[at+1:])
	} else {
		head = ref
	}

	var version string
	var contractsHead, contractsTail []string

	if tail != "" {
		colonTail := strings.Index(tail, ":")
		if colonTail >= 0 {
			version = strings.TrimSpace(tail[:colonTail])
			contractsTail = splitContractList(tail[colonTail+1:])
		} else {
			version = tail
		}
	}

	colonHead := strings.Index(head, ":")
	if colonHead >= 0 {
		head = strings.TrimSpace(head[:colonHead])
		contractsHead = splitContractList(head[colonHead+1:])
	}
	projectKey := strings.TrimSpace(head)
	if projectKey == "" {
		return Ref{}, fmt.Errorf("empty project key in ref")
	}

	contractsHead = append(contractsHead, contractsTail...)

	parsed = Ref{
		ProjectKey: projectKey,
		Contracts:  contractsHead,
		Version:    version,
	}
	return
}

func splitContractList(s string) (list []string) {

	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			list = append(list, part)
		}
	}
	return
}

// ResolveAlias подставляет алиас в projectKey: первый сегмент до точки заменяется по aliases.
func ResolveAlias(projectKey string, aliases map[string]string) (resolved string) {

	if projectKey == "" || len(aliases) == 0 {
		return projectKey
	}
	firstDot := strings.Index(projectKey, ".")
	if firstDot <= 0 {
		prefix, ok := aliases[projectKey]
		if ok {
			return prefix
		}
		return projectKey
	}
	alias := projectKey[:firstDot]
	if prefix, ok := aliases[alias]; ok {
		return prefix + "." + projectKey[firstDot+1:]
	}
	return projectKey
}
