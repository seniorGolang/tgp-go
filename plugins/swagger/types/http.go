// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package types

import (
	"fmt"
)

func CodeToText(code int) (result string) {
	var found bool
	if result, found = statusText[code]; found {
		return
	}
	return fmt.Sprintf("unknown error %d", code)
}

func IsValidHTTPCode(code int) (valid bool) {
	_, valid = statusText[code]
	return
}

var statusText = map[int]string{
	100: "Continue",
	101: "Switching Protocols",
	102: "Processing",
	200: "Successful operation",
	201: "Created",
	202: "Accepted",
	203: "Non-Authoritative Information",
	204: "No Content",
	205: "Reset Content",
	206: "Partial Content",
	207: "Multi-Status",
	208: "Already Reported",
	226: "IM Used",
	300: "Multiple Choices",
	301: "Moved Permanently",
	302: "Found",
	303: "See Other",
	304: "Not Modified",
	305: "Use Proxy",
	307: "Temporary Redirect",
	308: "Permanent Redirect",
	400: "Bad Request",
	401: "Unauthorized",
	402: "Payment Required",
	403: "Forbidden",
	404: "Not Found",
	405: "Method Not Allowed",
	406: "Not Acceptable",
	407: "Proxy Authentication Required",
	408: "Request Timeout",
	409: "Conflict",
	410: "Gone",
	411: "Length Required",
	412: "Precondition Failed",
	413: "Request Entity Too Large",
	414: "Request URI Too Long",
	415: "Unsupported Media Type",
	416: "Requested Range Not Satisfiable",
	417: "Expectation Failed",
	418: "I'm a teapot",
	422: "Unprocessable Entity",
	423: "Locked",
	424: "Failed Dependency",
	426: "Upgrade Required",
	428: "Precondition Required",
	429: "Too Many Requests",
	431: "Request Header Fields Too Large",
	451: "Unavailable For Legal Reasons",
	500: "Internal Server Error",
	501: "Not Implemented",
	502: "Bad Gateway",
	503: "Service Unavailable",
	504: "Gateway Timeout",
	505: "HTTP Version Not Supported",
	506: "Variant Also Negotiates",
	507: "Insufficient Storage",
	508: "Loop Detected",
	510: "Not Extended",
	511: "Network Authentication Required",
}

func JSONRPCSchemaPerPath(key string, schema Schema) (result Schema) {
	return Schema{
		Type: "object",
		Properties: Properties{
			"jsonrpc": Schema{
				Type:    "string",
				Example: "2.0",
			},
			"id": Schema{
				Type: "number",
			},
			key: schema,
		},
		Required: []string{"jsonrpc", key},
	}
}

func JSONRPCErrorSchema() (result Schema) {
	return Schema{
		Type: "object",
		Properties: Properties{
			"jsonrpc": Schema{
				Type:    "string",
				Example: "2.0",
			},
			"error": Schema{
				Type: "object",
				Properties: Properties{
					"code": Schema{
						Type: "number",
					},
					"message": Schema{
						Type: "string",
					},
					"data": Schema{
						Type: "object",
					},
				},
				Required: []string{"code", "message"},
			},
			"id": Schema{
				Type: "number",
			},
		},
		Required: []string{"jsonrpc", "error", "id"},
	}
}
