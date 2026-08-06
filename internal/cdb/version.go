package cdb

import (
	"regexp"
	"strings"
)

const maxVersionNameLength = 255

func NormalizeVersionName(version string) (s string) {

	if version == "" {
		return versionDefault
	}
	invalid := regexp.MustCompile(`[/\\:*?"<>|\s]+`)
	s = invalid.ReplaceAllString(version, "-")
	multi := regexp.MustCompile(`-+`)
	s = multi.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-.")
	if s == "" {
		return versionDefault
	}
	if len(s) > maxVersionNameLength {
		s = strings.TrimSuffix(s[:maxVersionNameLength], "-")
	}
	return s
}
