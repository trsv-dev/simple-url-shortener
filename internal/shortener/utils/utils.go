package utils

import "regexp"

// IsValidURL Проверка переданного URL на валидность (URL не пустой и соответствует требованиям к URL).
func IsValidURL(longURL string) bool {
	pattern := `^https?://([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}(/[^\s]*)?$`
	matched, _ := regexp.MatchString(pattern, longURL)
	return matched
}
