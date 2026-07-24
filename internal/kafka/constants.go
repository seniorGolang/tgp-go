// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package kafka

const (
	AcksAllISR = "allISRAcks"
	AcksLeader = "leaderAck"
	AcksNoAck  = "noAck"

	CodecJSON    = "json"
	CodecBytes   = "bytes"
	CodecMsgpack = "msgpack"
	CodecCBOR    = "cbor"
	CodecYAML    = "yaml"
	CodecXML     = "xml"
)

// BuiltinCodecNames возвращает имена встроенных кодеков.
func BuiltinCodecNames() (names []string) {

	return []string{
		CodecJSON,
		CodecBytes,
		CodecMsgpack,
		CodecCBOR,
		CodecYAML,
		CodecXML,
	}
}

// IsBuiltinCodec проверяет, является ли имя встроенным кодеком.
func IsBuiltinCodec(name string) (ok bool) {

	for _, builtin := range BuiltinCodecNames() {
		if name == builtin {
			return true
		}
	}
	return false
}
