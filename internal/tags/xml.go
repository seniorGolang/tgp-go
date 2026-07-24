package tags

import "strings"

// ExchangeXMLTag возвращает xml-тег exchange-поля с тем же wire-именем, что и json.
// Пустая строка — xml-тег не эмитить (inline / пустой json).
func ExchangeXMLTag(jsonTag string) (xmlTag string) {

	if jsonTag == "" || strings.Contains(jsonTag, "inline") {
		return ""
	}
	return jsonTag
}
