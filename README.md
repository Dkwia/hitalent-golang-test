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

