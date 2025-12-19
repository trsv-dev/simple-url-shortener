package utils

import "regexp"

var urlRegexp = regexp.MustCompile(
	`^https?://([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}(/[^\s]*)?$`,
)

// IsValidURL Проверка переданного URL на валидность (URL не пустой и соответствует требованиям к URL).
func IsValidURL(longURL string) bool {
	return urlRegexp.MatchString(longURL)
}
