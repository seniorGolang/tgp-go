// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package locale

import (
	"os"
	"strings"
	"sync"
)

type Language string

const (
	LanguageEN Language = "en"
	LanguageRU Language = "ru"
)

var (
	detectedLanguage   Language
	detectLanguageOnce sync.Once
)

// DetectLanguage читает TG_LANG; результат кэшируется после первого вызова.
func DetectLanguage() Language {

	detectLanguageOnce.Do(func() {
		tgLang := os.Getenv("TG_LANG")
		if tgLang == "" {
			detectedLanguage = LanguageEN
			return
		}

		tgLang = strings.ToLower(tgLang)
		if strings.HasPrefix(tgLang, "ru") {
			detectedLanguage = LanguageRU
			return
		}
		if strings.HasPrefix(tgLang, "en") {
			detectedLanguage = LanguageEN
			return
		}

		detectedLanguage = LanguageEN
	})
	return detectedLanguage
}
