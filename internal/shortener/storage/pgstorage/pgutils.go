package pgstorage

import (
	"fmt"
	"strings"
)

// ConvertDSNToPostgresURI формирует строку подключения в формате URI на основе DSN,
// переданного в конфигурации в виде строки key=value через пробел.
func ConvertDSNToPostgresURI(dbDSN string) string {
	// разбиваем строку с DSN базы данных на слайс строк
	dsnParts := strings.Split(dbDSN, " ")

	// создаем мапу слайсов строк
	creds := make(map[string]string)

	// итерируемся по слайсу DSN
	for _, part := range dsnParts {
		// разбиваем каждую часть слайса на пару значений (по `=`, т.е. ключ и значение)
		pair := strings.SplitN(part, "=", 2)
		// если длина пары равна 2
		if len(pair) == 2 {
			// добавляем в мапу
			creds[pair[0]] = pair[1]
		}
	}

	// формируем строку подключения в формате URI, которую понимает migrate
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		creds["user"], creds["password"], creds["host"], creds["port"], creds["dbname"], creds["sslmode"])
}
