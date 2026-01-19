// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package locale

import (
	"os"
	"strings"
	"sync"
)

// Language represents supported languages
type Language string

const (
	LanguageEN Language = "en"
	LanguageRU Language = "ru"
)

var (
	detectedLanguage   Language
	detectLanguageOnce sync.Once
)

// DetectLanguage detects language from TG_LANG environment variable.
// Returns "ru" if Russian locale is detected, "en" otherwise.
// Defaults to English if TG_LANG is not set or cannot be determined.
// Result is cached after first call (runOnce).
func DetectLanguage() Language {

	detectLanguageOnce.Do(func() {
		// Check TG_LANG environment variable
		tgLang := os.Getenv("TG_LANG")
		if tgLang == "" {
			// Default to English if TG_LANG is not set
			detectedLanguage = LanguageEN
			return
		}

		// Extract language code (e.g., "ru" -> "ru", "en" -> "en")
		tgLang = strings.ToLower(tgLang)
		if strings.HasPrefix(tgLang, "ru") {
			detectedLanguage = LanguageRU
			return
		}
		if strings.HasPrefix(tgLang, "en") {
			detectedLanguage = LanguageEN
			return
		}

		// Default to English if code cannot be determined
		detectedLanguage = LanguageEN
	})
	return detectedLanguage
}
