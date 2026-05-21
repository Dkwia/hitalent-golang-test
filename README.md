# API организационной структуры

REST API для работы с подразделениями и сотрудниками.

## Возможности

- создание подразделений и сотрудников;
- получение подразделения с деревом дочерних подразделений;
- перемещение подразделений;
- удаление подразделений в режимах `cascade` и `reassign`;
- валидация бизнес-ограничений;
- миграции через `goose`;
- запуск в Docker / Docker Compose.

## Стек

- Go
- `net/http`
- GORM
- PostgreSQL
- goose

## Запуск

```bash
docker-compose up --build
```

Сервис будет доступен на `http://localhost:8080`.

## Переменные окружения

- `PORT` — порт приложения, по умолчанию `8080`
- `DB_HOST` — хост PostgreSQL
- `DB_PORT` — порт PostgreSQL
- `DB_USER` — пользователь базы
- `DB_PASSWORD` — пароль базы
- `DB_NAME` — имя базы
- `DB_SSLMODE` — режим SSL, по умолчанию `disable`
- `DB_TIMEZONE` — часовой пояс, по умолчанию `UTC`

## Эндпоинты

- `POST /departments/`
- `POST /departments/{id}/employees/`
- `GET /departments/{id}?depth=1&include_employees=true`
- `PATCH /departments/{id}`
- `DELETE /departments/{id}?mode=cascade`
- `DELETE /departments/{id}?mode=reassign&reassign_to_department_id=...`

## Миграции

SQL-миграции лежат в папке `migrations/` и применяются при старте приложения.

## Тесты

```bash
go test ./...
```
PI организационной структуры

REST API для работы с подразделениями и сотрудниками по тестовому заданию.

## Возможности

- создание подразделений;
- создание сотрудников внутри подразделения;
- получение подразделения с сотрудниками и древовидной структурой дочерних подразделений;
- перемещение подразделения в другое подразделение;
- удаление подразделения в режимах `cascade` и `reassign`;
- валидация бизнес-ограничений;
- миграции через `goose`;
- запуск через Docker и `docker-compose`.

## Стек

- Go
- `net/http`
- GORM
- PostgreSQL
- goose
- Docker / Docker Compose

## Структура проекта

```text
cmd/api            — точка входа приложения
internal/config     — загрузка конфигурации из окружения
internal/db         — подключение к PostgreSQL
internal/handler    — HTTP-обработчики
internal/service    — бизнес-логика
internal/repository — доступ к БД через GORM
internal/models     — модели GORM
internal/httputil   — JSON-утилиты
migrations          — SQL-миграции goose
```

## Переменные окружения

Создай файл `.env` на основе `.env.example`.

### Для приложения

- `PORT` — порт HTTP-сервера внутри контейнера, по умолчанию `8080` (если не задан, используется `8080`)
- `DB_HOST` — хост PostgreSQL, по умолчанию `db`
- `DB_PORT` — порт PostgreSQL внутри сети Docker, по умолчанию `5432`
- `DB_USER` — пользователь базы, по умолчанию `postgres`
- `DB_PASSWORD` — пароль базы, по умолчанию `postgres`
- `DB_NAME` — имя базы, по умолчанию `organization`
- `DB_SSLMODE` — режим SSL, по умолчанию `disable`
- `DB_TIMEZONE` — часовой пояс, по умолчанию `UTC`

### Для docker-compose

- `APP_PORT` — внешний порт приложения на хосте, например `8088`
- `DB_PORT` — внешний порт PostgreSQL на хосте, например `5433`

## Запуск

```bash
cp .env.example .env
docker compose up --build
```

После запуска API будет доступно на хостовом порту из `APP_PORT`.

Пример:

```bash
curl http://localhost:8088/healthz
```

## Проверка здоровья

```bash
curl http://localhost:8088/healthz
```

Ответ:

```text
ok
```

## API

### 1. Создать подразделение

`POST /departments/`

Body:

```json
{
  "name": "Backend",
  "parent_id": null
}
```

Ответ: `201 Created`

---

### 2. Создать сотрудника в подразделении

`POST /departments/{id}/employees/`

Body:

```json
{
  "full_name": "Ivan Ivanov",
  "position": "Go Developer",
  "hired_at": "2025-01-01"
}
```

Ответ: `201 Created`

---

### 3. Получить подразделение с деревом

`GET /departments/{id}?depth=1&include_employees=true`

Параметры:

- `depth` — глубина вложенности, от `0` до `5`
- `include_employees` — `true` или `false`

Ответ: `200 OK`

---

### 4. Обновить подразделение

`PATCH /departments/{id}`

Body:

```json
{
  "name": "Platform",
  "parent_id": 1
}
```

Ответ: `200 OK`

---

### 5. Удалить подразделение

`DELETE /departments/{id}?mode=cascade`

или

`DELETE /departments/{id}?mode=reassign&reassign_to_department_id=2`

Ответ: `204 No Content`

## Бизнес-правила

- нельзя создать сотрудника в несуществующем подразделении;
- `name` подразделения должен быть непустым и длиной 1..200;
- `full_name` сотрудника должен быть непустым и длиной 1..200;
- `position` сотрудника должен быть непустым и длиной 1..200;
- в пределах одного родителя названия подразделений должны быть уникальны;
- нельзя сделать подразделение родителем самого себя;
- нельзя создать цикл в дереве подразделений;
- при `cascade` удаление выполняется каскадно;
- при `reassign` сотрудники переводятся в указанное подразделение.

## Миграции

Миграции лежат в папке `migrations/` и применяются автоматически при старте приложения через `goose`.

## Тесты

Запуск тестов:

```bash
go test ./...
```

Для локальной проверки можно также использовать скрипт:

```bash
./run_tests.sh
```

## Примеры запросов

Создать root-подразделение:

```bash
curl -X POST http://localhost:8088/departments/ \
  -H "Content-Type: application/json" \
  -d '{"name":"Backend"}'
```

Создать сотрудника:

```bash
curl -X POST http://localhost:8088/departments/1/employees/ \
  -H "Content-Type: application/json" \
  -d '{"full_name":"Ivan Ivanov","position":"Go Developer"}'
```

Получить дерево подразделения:

```bash
curl "http://localhost:8088/departments/1?depth=5&include_employees=true"
```

## Примечание

Если порт `8080` или `5432` уже занят на хосте, поменяй `APP_PORT` и `DB_PORT` в `.env`.

