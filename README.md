# Simple URL Shortener

**Simple URL Shortener** — это backend-сервис для сокращения URL, реализованный на Go.  
Сервис предоставляет API для генерации коротких ссылок, их восстановления, работы с пользовательскими сессиями, а также поддержки хранения данных в памяти, в файле и в PostgreSQL.

Проект построен с использованием модульной архитектуры, покрыт тестами, поддерживает GZIP-сжатие, авторизацию через cookie и CLI-конфигурацию.

---

## ⚙️ Возможности

- 🔗 Сокращение длинных URL
- 🔄 Восстановление оригинальных URL
- 👤 Авторизация и сессии через cookie
- 🌐 HTTP API и gRPC API
- 🎛️ Независимый запуск HTTP сервера и gRPC сервера с настройкой порта
- 🧠 Три варианта хранения - в оперативной памяти, файле и PostgreSQL
- 💥 Проверка сгенерированных коротких URL на коллизии
- ⬅️ Удаление пользовательских ссылок
- ⚙️ Конфигурация через флаги и переменные окружения
- 📉 Поддержка GZIP-сжатия
- 📋 Логирование событий с задаваемым уровнем
- 🧪 Покрытие бизнес-логики тестами

---

## 🧬 Структура проекта

```
cmd/
└── shortener/             # Точка входа (main.go)

internal/
├── shortener/             # Основная бизнес-логика
│   ├── auth/              # Логика работы с JWT токенами
│   ├── config/            # Конфигурация сервиса
│   ├── contextkeys/       # Ключи для контекста
│   ├── errs/              # Кастомные ошибки
│   ├── generator/         # Генератор коротких ссылок
│   ├── handlers/          # HTTP-обработчики
│   ├── grpc/              # gRPC сервер
│   │   ├── handlers/      # gRPC хендлеры
│   │   ├── interceptors/  # gRPC интерсепторы
│   │   ├── proto/         # protobuf-описания и сгенерированный код
│   │   └── server/        # Логика запуска gRPC сервера
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

# Адрес и порт сервера
SERVER_ADDRESS=localhost:8080
# Базовый URL сервера
BASE_URL=http://localhost:8080
# Включить логирование
LOGGING=true
# Возможные варианты - Debug, Info, Warn, Error, DPanic, Panic, Fatal
LOG_LEVEL=Debug
# По умолчанию будет создан и использован "/tmp/short-url-storage.json"
FILE_STORAGE_PATH=
# Если не задан DATABASE_DSN - будет использовано файловое хранилище
# Пример: 'host=localhost port=5432 user=db_user password=xxxxx dbname=db_name sslmode=disable'
DATABASE_DSN=
# Секретный ключ для подписи JWT-токенов
SECRET_KEY=SuperSecretKey
# Время жизни JWT-токена. Можно указывать в формате TOKEN_EXP=24h, TOKEN_EXP=30m, TOKEN_EXP=3600s
TOKEN_EXP=24h
# JSON-файл с параметрами конфигурации
CONFIG=
# Доверенная подсеть (CIDR)
TRUSTED_SUBNET=127.0.0.0/8
# Запуск gRPC сервера
ENABLE_GRPC=false
# gRPC порт
GRPC_PORT=9999
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
# Запуск с БД и только HTTP сервером
./shortener -l=true -ll Debug -e 12h -k UIfuBer431qY1 -d 'host=localhost port=5432 user=username password=user_password dbname=user_dbname sslmode=disable' -t 192.168.1.0/24
# Запуск с файловым хранилищем и только HTTP сервером
./shortener -l=true -ll Debug -e 12h -k UIfuBer431qY1 -f "/tmp/short-url-storage-filedb.json" -t 192.168.1.0/24
# Запуск с хранилищем в памяти и только HTTP сервером
./shortener -l=true -ll Debug -e 12h -k UIfuBer431qY1 -f "" -t 192.168.1.0/24
# Запуск с БД и только gRPC сервером
./shortener -l=true -ll Debug -e 12h -k UIfuBer431qY1 -d 'host=localhost port=5432 user=username password=user_password dbname=user_dbname sslmode=disable' -t 192.168.1.0/24 -grpc=true -gp=9999 -http=false
```

Описание флагов:

```bash
-a  Адрес и порт HTTP-сервера (по умолчанию "localhost:8080")
-b  Базовый URL сервиса (по умолчанию "http://localhost:8080")
-d  Строка подключения к PostgreSQL (пример: `host=localhost port=5432 user=db_user password=xxxxx dbname=db_name sslmode=disable`)
-e  Время действия JWT-токена. Пример: 24h, 45m, 60s и т.д. (по умолчанию 24h0m0s)
-f  Путь до файла хранения (по умолчанию используется "/tmp/short-url-storage.json")
-k  Секретный ключ для подписи JWT-токена. Пример: UIfuBer431qY1crEUgzIem9 (по умолчанию "Change_Me_On_Production")
-l  Включение логгирования (по умолчанию true)
-ll Выбор уровня логгирования (по умолчанию Info, возможные варианты - Debug, Info, Warn, Error, DPanic, Panic, Fatal)
-f JSON-файл с параметрами конфигурации
-t Доверенная подсеть (CIDR)
-grpc Включение gRPC сервера (по умолчанию false)
-gp Порт gRPC сервера (по умолчанию 9999)
-http Включение HTTP сервера (по умолчанию true)
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
| GET    | `/api/internal/stats`| Количество пользователей и сокращённых URL в сервисе |
| DELETE | `/api/user/urls`     | Удаление URL (асинхронно)         |
| GET    | `/ping`              | Проверка доступности хранилища    |

---

## 🧰 Используемые технологии

- 🐹 go 1.26.0+
- 🐘 PostgreSQL
- 🌐 HTTP API
- 📡 gRPC API
- 🧩 Protocol Buffers (protobuf)
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

## 🍪 Аутентификация HTTP-запросов

Для аутентификации пользователей используется JWT (JSON Web Token), передаваемый в cookie JWT.

При отсутствии cookie сервер генерирует новый JWT и устанавливает cookie с заданным сроком действия. 
Затем сервер проверяет токен, извлекает идентификатор пользователя (UserID) и добавляет его в контекст запроса. 
Если токен недействителен или произошла ошибка при его обработке, запрос отклоняется.
Токен подписывается с использованием алгоритма HS256 и секретного ключа. Срок его действия задаётся конфигурацией приложения.

## 🔐 Аутентификация gRPC-запросов

Для аутентификации пользователей используется JWT (JSON Web Token), передаваемый в gRPC metadata в поле authorization с префиксом Bearer.
При получении gRPC-запроса сервер извлекает токен из metadata, проверяет его подпись и срок действия, после чего определяет идентификатор пользователя (UserID). 
Если токен отсутствует или недействителен, запрос отклоняется.

JWT создаётся сервером и содержит идентификатор пользователя (UserID) и время истечения срока действия (ExpiresAt). 
Подпись токена формируется с использованием алгоритма HS256 и секретного ключа.

Клиент должен передавать токен в формате: **authorization: Bearer <JWT>**
Сервер извлекает токен из входящих metadata, проверяет его подпись и срок действия, после чего определяет идентификатор пользователя. 

## 🔑 Получение JWT-токена

Для получения JWT-токена предусмотрен gRPC-метод `GetToken`.
Метод вызывается без обязательной авторизации и не принимает параметры. 
В ответ сервер возвращает JWT-токен, который необходимо передавать в последующих gRPC-запросах через metadata `authorization` в формате `Bearer <JWT>`.
Если в запросе `GetToken` уже присутствует заголовок `authorization`, сервер возвращает ошибку `AlreadyExists`.

**Пример запроса:**
```protobuf
{}
```

**Пример ответа:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

Полученный токен используется для аутентификации последующих gRPC-запросов. Срок действия токена определяется конфигурацией приложения.

## 🧶 Работа с HTTP эндпоинтами
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
- `403 Forbidden` — неправильный IP пользователя, список доверенных подсетей пуст или ip пользователя не входит в доверенную подсеть
- `200 OK` — список получен
- `401 Unauthorized` — пользователь неавторизован

---

## 👤 Статистика по кол-ву пользователей и ссылок (доверенная подсеть)

**GET** `/api/internal/stats`  
Количество пользователей и сокращённых URL в сервисе.

### 📥 Возможные ответы:
```text
HTTP/1.1 200 OK
Content-Type: application/json

[
    {
      "urls": <int>, // количество сокращённых URL в сервисе
      "users": <int> // количество пользователей в сервисе
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

## 🔌 Работа с gRPC-эндпоинтами

<details>
<summary>Посмотреть детали</summary>

### 🎟️ Получение JWT-токена

**RPC** `GetToken`

Получает JWT-токен для последующей аутентификации gRPC-запросов. Метод вызывается без параметров и не требует предварительной авторизации.
#### 📤 Пример запроса:

```json
{}
```

#### 📥 Пример ответа:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### 📥 Возможные ответы:
- `OK` — токен успешно создан
- `AlreadyExists` — в запросе уже присутствует заголовок `authorization`
- `Internal` — ошибка генерации токена

Полученный токен передаётся в metadata последующих запросов:

```text
authorization: Bearer <JWT>
```

---

### 🔗 Сокращение URL

**RPC** `Shorten`
Создаёт короткую ссылку по оригинальному URL.

#### 📤 Пример запроса:
```json
{
  "original_url": "https://example.com/long-url"
}
```

#### 📥 Пример ответа:
```json
{
  "shortened_url": "http://localhost:8080/abc123"
}
```

#### 📥 Возможные ответы:
- `OK` — ссылка успешно сокращена
- `AlreadyExists` — оригинальный URL уже существует; в сообщении ошибки возвращается соответствующая короткая ссылка
- `InvalidArgument` — передан некорректный URL
- `Internal` — внутренняя ошибка сервера

---

### ✂️ Сокращение URL через text/plain

**RPC** `Post`
Создаёт короткую ссылку по оригинальному URL. В отличие от HTTP-эндпоинта, запрос передаётся в формате protobuf, а не `text/plain`.

#### 📤 Пример запроса:

```json
{
  "original_url": "https://example.com/long-url"
}
```

#### 📥 Пример ответа:
```json
{
  "shortened_url": "http://localhost:8080/abc123"
}
```

#### 📥 Возможные ответы:

- `OK` — ссылка успешно сокращена
- `AlreadyExists` — оригинальный URL уже существует; короткая ссылка возвращается в ответном сообщении вместе с ошибкой
- `InvalidArgument` — передан некорректный URL

---

### ↩️ Получение оригинального URL

**RPC** `GetOriginalURL`

Получает оригинальный URL по идентификатору короткой ссылки. В отличие от HTTP-эндпоинта `GET /{shortID}`, gRPC-метод не выполняет редирект, а возвращает оригинальный URL в ответе.

#### 📤 Пример запроса:
```json
{
  "short_url": "abc123"
}
```

#### 📥 Пример ответа:
```json
{
  "original_url": "https://example.com/long-url"
}
```

#### 📥 Возможные ответы:
- `OK` — оригинальный URL найден
- `NotFound` — ссылка помечена на удаление
- `InvalidArgument` — ошибка получения URL или некорректный идентификатор

---

### 🧺 Пакетное сокращение ссылок

**RPC** `ShortenBatch`
Позволяет сократить сразу несколько URL одним запросом.

#### 📤 Пример запроса:
```json
{
  "urls": [
    {
      "correlation_id": "1",
      "original_url": "https://example.com/1"
    },
    {
      "correlation_id": "2",
      "original_url": "https://example.com/2"
    }
  ]
}
```

#### 📥 Пример ответа:
```json
{
  "urls": [
    {
      "correlation_id": "1",
      "original_url": "https://example.com/1",
      "short_url": "abc123"
    },
    {
      "correlation_id": "2",
      "original_url": "https://example.com/2",
      "short_url": "xyz456"
    }
  ]
}
```

#### 📥 Возможные ответы:
- `OK` — ссылки успешно обработаны
- `InvalidArgument` — передан пустой пакет
- `Internal` — ошибка пакетного сохранения ссылок

---

### 👤 Получение ссылок пользователя
**RPC** `GetUserURLs`

Возвращает список сокращённых ссылок текущего пользователя. Идентификатор пользователя извлекается из JWT интерсептором.

#### 📤 Пример запроса:
```json
{}
```

#### 📥 Пример ответа:
```json
{
  "user_urls": [
    {
      "original_url": "https://example.com",
      "short_url": "abc123"
    }
  ]
}
```

#### 📥 Возможные ответы:
- `OK` — список получен
- `Unauthenticated` — отсутствует или недействителен JWT
- `Internal` — ошибка получения ссылок

---

### 🗑️ Удаление ссылок пользователя

**RPC** `DeleteURLs`
Асинхронно помечает переданные короткие ссылки на удаление.

#### 📤 Пример запроса:
```json
{
  "short_urls": [
    "abc123",
    "xyz456"
  ]
}
```

#### 📥 Пример ответа:
```json
{}
```

#### 📥 Возможные ответы:
- `OK` — запрос на удаление обработан
- `Unauthenticated` — отсутствует или недействителен JWT
- `Internal` — ошибка при удалении ссылок

---

### 📊 Статистика пользователей и ссылок

**RPC** `Stats`
Возвращает общее количество пользователей и сокращённых URL. Доступ разрешён только из доверенной подсети, заданной через переменную окружения `TRUSTED_SUBNET` или флаг `-t`.

Для проверки IP клиента используется metadata `x-real-ip`.

#### 📤 Пример запроса:
```json
{}
```

#### 📥 Пример ответа:
```json
{
  "users_amount": 10,
  "urls_amount": 25
}
```

#### 📥 Возможные ответы:
- `OK` — статистика получена
- `PermissionDenied` — доступ запрещён или IP не входит в доверенную подсеть
- `InvalidArgument` — отсутствует metadata или заголовок `x-real-ip`
- `Internal` — ошибка получения статистики или некорректная конфигурация подсети

Если доверенная подсеть не задана, доступ к статистике запрещён.

---

### ❤️ Проверка соединения с БД

**RPC** `PingDB`
Проверяет доступность базы данных.

#### 📤 Пример запроса:
```json
{}
```

#### 📥 Пример ответа:
```json
{}
```

#### 📥 Возможные ответы:
- `OK` — соединение с БД работает
- `Internal` — ошибка соединения с БД

</details>