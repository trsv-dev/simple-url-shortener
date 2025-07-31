# Simple URL Shortener

**Simple URL Shortener** — это backend-сервис для сокращения URL, реализованный на Go.  
Сервис предоставляет API для генерации коротких ссылок, их восстановления, работы с пользовательскими сессиями, а также поддержки хранения данных в памяти, в файле и в PostgreSQL.

Проект построен с использованием модульной архитектуры, покрыт тестами, поддерживает GZIP-сжатие, авторизацию через cookie и CLI-конфигурацию.

---

## ⚙️ Возможности

- 🔗 Сокращение длинных URL
- 🔄 Восстановление оригинальных URL
- 👤 Авторизация и сессии через cookie
- 🧠 Три варианта хранения - в оперативной памяти, файле и PostgreSQL
- 💥 Проверка сгенерированных коротких URL на коллизии
- ⬅️ Удаление пользовательских ссылок
- ⚙️ Конфигурация через флаги и переменные окружения
- 📉 Поддержка GZIP-сжатия
- 📋 Логгирование событий с задаваемым уровнем
- 🧪 Покрытие бизнес-логики тестами

---

## 🧬 Структура проекта

```
cmd/
└── shortener/             # Точка входа (main.go)

internal/
├── shortener/             # Основная бизнес-логика
│   ├── config/            # Конфигурация сервиса
│   ├── contextkeys/       # Ключи для контекста
│   ├── errs/              # Кастомные ошибки
│   ├── generator/         # Генератор коротких ссылок
│   ├── handlers/          # HTTP-обработчики
│   ├── logger/            # Инициализация логгера
│   ├── middleware/        # Мидлвары: GZIP, авторизация
│   ├── mocks/             # Моки для тестов
│   ├── models/            # Модели данных
│   ├── router/            # Роутинг
│   ├── server/            # Запуск сервера
│   ├── storage/           # Реализация хранилищ
│   │   ├── file/          # Файловое хранилище
│   │   ├── memory/        # Хранилище в памяти
│   │   └── pgstorage/     # PostgreSQL-хранилище
│   └── utils/             # Вспомогательные функции

go.mod, go.sum            # Зависимости
README.md                 # Документация
```

---

## 🚀 Быстрый старт

### 1. Клонируйте репозиторий

```bash
git clone https://github.com/trsv-dev/simple-url-shortener.git
cd simple-url-shortener
```

### 2. Соберите бинарник

```bash
cd cmd/shortener
go build -o shortener
```

### 3. Если собираетесь использовать хранение URL в Postgres - cоздайте БД в Postgres
```bash
psql -h localhost -U postgres
postgres=# create database dbname; 
postgres=# create user username with encrypted password 'userpassword'; 
postgres=# grant all privileges on database dbname to username;
postgres=# alter database dbname owner to username;
```

### 4. Настройте переменные окружения или используйте флаги запуска

```env
# Пример .env. Вам нужно заполнить своими данными

SERVER_ADDRESS=localhost:8080
BASE_URL=http://localhost:8080
LOGGING=true
# Возможные варианты - Debug, Info, Warn, Error, DPanic, Panic, Fatal
LOG_LEVEL=Debug
# По умолчанию будет создан и использован "/tmp/short-url-storage.json"
FILE_STORAGE_PATH=
# Если не задан DATABASE_DSN - будет использовано файловое хранилище
# Пример: 'host=localhost port=5432 user=db_user password=xxxxx dbname=db_name sslmode=disable'
DATABASE_DSN=
```

### 5. Запустите приложение

```bash
./shortener
```

Или напрямую из Go:

```bash
go run cmd/shortener/main.go
```

### 🔧 Пример запуска через скомпилированный бинарник с флагами:

```bash
# Запуск с БД
./shortener -l=true -ll Debug -d 'host=localhost port=5432 user=username password=user_password dbname=user_dbname sslmode=disable'
# Запуск с файловым хранилищем
./shortener -l=true -ll Debug -f "/tmp/short-url-storage-filedb.json"
# Запуск с хранилищем в памяти
./shortener -l=true -ll Debug -f ""
```

Описание флагов:

```bash
-a  Адрес и порт HTTP-сервера (по умолчанию "localhost:8080")
-b  Базовый URL сервиса (по умолчанию "http://localhost:8080")
-d  Строка подключения к PostgreSQL (пример: `host=localhost port=5432 user=db_user password=xxxxx dbname=db_name sslmode=disable`)
-f  Путь до файла хранения (по умолчанию используется "/tmp/short-url-storage.json")
-l  Включение логгирования (по умолчанию true)
-ll Выбор уровня логгирования (по умолчанию Info, возможные варианты - Debug, Info, Warn, Error, DPanic, Panic, Fatal)
```

---

## 🔐 API-эндпоинты

| Метод  | Путь                 | Назначение                        |
|--------|----------------------|---------------------------------|
| POST   | `/`                  | Сокращение URL (plain text)      |
| POST   | `/api/shorten`       | Сокращение URL (JSON API)         |
| POST   | `/api/shorten/batch` | Пакетная загрузка URL             |
| GET    | `/{shortID}`         | Восстановление оригинального URL |
| GET    | `/api/user/urls`     | Получение всех URL пользователя   |
| DELETE | `/api/user/urls`     | Удаление URL (асинхронно)         |
| GET    | `/ping`              | Проверка доступности хранилища    |

---

## 🧰 Используемые технологии

- 🐹 go 1.22.0+
- 🐘 PostgreSQL
- 🍪 Cookie-based аутентификация
- 🔐 GZIP middleware
- 🧪 Testify, sqlmock

---

## 🧱 Архитектура

- **handlers** — HTTP-обработчики
- **middleware** — авторизация, сжатие и логирование
- **storage** — интерфейсы и реализации хранилищ
- **generator** — генерация коротких ID
- **config** — загрузка конфигурации
- **router** — маршрутизация запросов

## 🧶 Работа с эндпоинтами
<details>
<summary>Посмотреть детали</summary>

## 🔗 Сокращение URL

**POST** `/api/shorten`  
Создаёт короткую ссылку по оригинальному URL (JSON).

### 📤 Пример запроса:
```text
POST /api/shorten HTTP/1.1
Content-Type: application/json

{
  "url": "https://example.com/long-url"
}
```

### 📥 Возможные ответы:
- `405 Method Not Allowed` — если метод не POST
- `201 Created` — ссылка сокращена
- `400 Bad Request` — некорректный формат
- `409 Conflict` — уже существует
- `500 Internal Server Error` — ошибка сервера

---

## ✂️ Сокращение URL через text/plain

**POST** `/`  
Передаёт URL в теле запроса (не JSON).

### 📤 Пример запроса:
```text
POST / HTTP/1.1
Content-Type: text/plain

https://example.com/long-url
```

### 📥 Возможные ответы:
- `201 Created` — ссылка сокращена
- `400 Bad Request` — невалидный формат
- `409 Conflict` — уже есть в базе
- `500 Internal Server Error` — ошибка сервера

---

## ↩️ Получение оригинального URL

**GET** `/{shortID}`  
Редирект на оригинальный URL.

### 📥 Возможные ответы:
- `307 Temporary Redirect` — редирект
- `410 Gone` — ссылка помечена на удаление, но еще не удалена
- `404 Not Found` — не найдена

---

## 🧺 Пакетное сокращение ссылок

**POST** `/api/shorten/batch`  
Позволяет сократить сразу несколько URL.

### 📤 Пример запроса:
```text
POST /api/shorten/batch HTTP/1.1
Content-Type: application/json

[
  {
    "correlation_id": "1",
    "original_url": "https://example.com/1"
  },
  {
    "correlation_id": "2",
    "original_url": "https://example.com/2"
  }
]
```

### 📥 Возможные ответы:
- `405 Method Not Allowed` — если метод не POST
- `201 Created` — ссылки созданы
- `400 Bad Request` — ошибка формата
- `500 Internal Server Error` — ошибка сервера

---

## 👤 Получение ссылок пользователя

**GET** `/api/user/urls`  
Список всех сокращённых ссылок пользователя.

### 📥 Возможные ответы:
```text
HTTP/1.1 200 OK
Content-Type: application/json

[
  {
    "short_url": "http://localhost:8080/abc123",
    "original_url": "https://example.com"
  }
]
```

- `405 Method Not Allowed` — если метод не GET
- `200 OK` — список получен
- `204 No Content` — пустой список
- `401 Unauthorized` — пользователь неавторизован

---

## 🗑 Удаление ссылок пользователя

**DELETE** `/api/user/urls`  
Асинхронное удаление ссылок.

### 📤 Пример запроса:
```text
DELETE /api/user/urls HTTP/1.1
Content-Type: application/json

["abc123", "xyz456"]
```

### 📥 Возможные ответы:
- `405 Method Not Allowed` — если метод не DELETE
- `202 Accepted` — удаление запущено
- `400 Bad Request` — некорректный формат
- `401 Unauthorized` — пользователь неавторизован
- `500 Internal Server Error` — ошибка сервера

---

## ❤️ Проверка сервера

**GET** `/ping`  
Проверяет доступность сервера и БД.

### 📥 Возможные ответы:
- `405 Method Not Allowed` — если метод не GET
- `200 OK` — работает
- `500 Internal Server Error` — БД недоступна

</details>