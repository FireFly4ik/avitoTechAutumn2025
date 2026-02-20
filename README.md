# PR Reviewer Assignment Service

Микросервис для автоматического назначения ревьюверов на Pull Request'ы.  
Управляет командами разработчиков, отслеживает активность участников и обеспечивает бесперебойное code-review даже при уходе сотрудников — автоматически переназначает ревью на активных коллег.

---

## Возможности

- **Автоматическое назначение** — при создании PR сервис выбирает до 2 ревьюверов из команды автора (случайным образом среди активных участников, исключая автора).
- **Управление командами** — создание команд, добавление участников с указанием статуса активности.
- **Деактивация участников** — поштучная или массовая деактивация с автоматическим переназначением открытых ревью.
- **Переназначение ревьюверов** — ручная замена конкретного ревьювера или массовая замена всех неактивных ревьюверов PR.
- **Идемпотентность** — повторный merge PR не вызывает ошибку, а возвращает текущее состояние.
- **Аутентификация** — Bearer-токены для ролей `admin` (запись) и `user` (чтение).
- **Мониторинг** — Prometheus-метрики, Grafana-дашборды, структурированные JSON-логи (Loki + Promtail).
- **Graceful shutdown** — корректное завершение HTTP-сервера и закрытие соединения с БД.
- **Database retry** — повторные попытки подключения к БД при старте (актуально для Docker).
- **Config validation** — проверка обязательных переменных окружения при запуске.

---

## Технологический стек

| Категория | Технология |
|-----------|-----------|
| Язык | Go 1.24 |
| HTTP | Gin |
| ORM | GORM + pgx/v5 |
| Миграции | golang-migrate |
| Логирование | zerolog (JSON) |
| БД | PostgreSQL 16 |
| Метрики | Prometheus client → Grafana |
| Логи в Grafana | Loki + Promtail |
| Контейнеризация | Docker (multi-stage build) + Docker Compose |
| Тесты | testify + mockery |
| Линтинг | golangci-lint (28 линтеров) |

---

## Архитектура

Проект построен по принципам **Clean Architecture** с чётким разделением на слои:

```
HTTP Request
     │
 Middleware   ← auth, logging, metrics, recovery, CORS
     │
 Handlers     ← валидация, маппинг типизированных request/response
     │
 Service      ← бизнес-логика, транзакции (TxManager)
     │
 Storage      ← репозитории (GORM), Transaction Manager
     │
 PostgreSQL
```

**Паттерны:** Repository, Dependency Injection, Transaction Manager (Do/DoRead), Domain Errors.

### Структура проекта

```
├── cmd/api/main.go              # Точка входа
├── internal/
│   ├── api/
│   │   ├── handlers/            # HTTP-обработчики + типизированные mappers
│   │   ├── middleware/          # Auth, Logger, Metrics, Recovery, CORS
│   │   └── server/              # HTTP-сервер с graceful shutdown
│   ├── config/                  # Конфигурация из env + Validate()
│   ├── domain/                  # Модели, интерфейсы, доменные ошибки
│   ├── service/                 # Бизнес-логика (team, user, pullRequest)
│   ├── storage/gorm/            # GORM-репозитории + TxManager
│   ├── logger/                  # zerolog setup
│   ├── metrics/                 # Prometheus collectors
│   └── mocks/                   # Моки (mockery)
├── migrations/                  # SQL миграции (up + down)
├── tests/
│   ├── unit/                    # Unit-тесты (handlers, middleware, service, config)
│   ├── integration/             # Интеграционные тесты (реальная БД)
│   └── e2e/                     # End-to-end тесты (полный API-workflow)
├── grafana/                     # Дашборды + provisioning (Prometheus, Loki)
├── prometheus/                  # Конфиг Prometheus + алерты
├── docker-compose.yml           # Production-окружение
├── docker-compose.test.yml      # Тестовое окружение
├── Dockerfile                   # Multi-stage build (golang:1.24-alpine → alpine:3.19)
├── Makefile                     # Команды управления
└── openapi.yml                  # OpenAPI 3.0 спецификация
```

---

## Быстрый старт

### Требования

- Docker 20.10+ и Docker Compose 1.29+
- Go 1.24+ (для локальной разработки и тестов)

### Запуск

```bash
git clone https://github.com/FireFly4ik/avitoTechAutumn2025.git
cd avitoTechAutumn2025

# Создать .env по примеру
cp .env.example .env

# Запустить всё (API + PostgreSQL + Prometheus + Grafana + Loki)
docker-compose up -d
```

### Проверка

```bash
curl http://localhost:8080/metrics
```

### Сервисы

| Сервис | URL | Credentials |
|--------|-----|-------------|
| API | http://localhost:8080 | Bearer `admin` / `user` |
| Swagger UI | http://localhost:8088 | — |
| Grafana | http://localhost:3000 | admin / admin |
| Prometheus | http://localhost:9090 | — |
| PostgreSQL | localhost:5433 | avito / avito_password |

---

## API

Полная спецификация — [`openapi.yml`](./openapi.yml).

### Эндпоинты

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| POST | `/team/add` | — | Создать команду с участниками |
| GET | `/team/get?team_name=` | user | Получить информацию о команде |
| POST | `/team/deactivate` | admin | Деактивировать участников команды |
| POST | `/users/setIsActive` | admin | Изменить статус активности пользователя |
| GET | `/users/getReview?user_id=` | user | PR, назначенные на пользователя |
| POST | `/pullRequest/create` | admin | Создать PR с автоназначением ревьюверов |
| POST | `/pullRequest/merge` | admin | Смержить PR (идемпотентно) |
| POST | `/pullRequest/reassign` | admin | Переназначить одного ревьювера |
| POST | `/pullRequest/reassignInactive` | admin | Переназначить всех неактивных ревьюверов PR |
| GET | `/metrics` | — | Prometheus-метрики |
| GET | `/openapi.yml` | — | OpenAPI 3.0 спецификация |

---

## Тестирование

```bash
# Unit-тесты (быстро, без зависимостей)
make test-unit

# Unit-тесты с покрытием
make test-unit-cover

# Интеграционные тесты (автоматически поднимает тестовую БД)
make test-integration

# E2E тесты (автоматически поднимает тестовое окружение)
make test-e2e

# Все тесты последовательно
make test-all
```

### Структура тестов

- **Unit** — сервисный слой с моками репозиториев и TxManager, обработчики с моками сервиса, middleware, config validation.
- **Integration** — реальная PostgreSQL через Docker, проверка всех слоёв вместе, изоляция через отдельную БД.
- **E2E** — полный API-workflow через HTTP-запросы: создание команды → создание PR → деактивация → переназначение → merge.

---

## Мониторинг

После `docker-compose up -d` доступны:

- **Prometheus** (http://localhost:9090) — сбор метрик `http_requests_total`, `http_request_duration_seconds`, `db_queries_total`, `service_operations_total`.
- **Grafana** (http://localhost:3000) — преднастроенные дашборды:
  - HTTP Performance — RPS, latency, status codes
  - Database Performance — query time
  - Service Teams / Pull Requests — бизнес-метрики
  - Application Logs — real-time JSON-логи через Loki

Логи пишутся в JSON-формате одновременно в stdout (Docker logs) и в файл `./logs/app.log` (доступен на хосте через volume).

---

## Make-команды

```bash
make help              # Список всех команд
make docker-up         # Запуск production
make docker-down       # Остановка
make test-unit         # Unit-тесты
make test-all          # Все тесты
make lint              # Линтер (golangci-lint)
make pre-commit        # fmt + lint + unit-тесты
make mocks             # Генерация моков
```

---

## Принятые решения

| Вопрос | Решение |
|--------|---------|
| Аутентификация | Bearer-токены (`ADMIN_TOKEN`, `USER_TOKEN`) — достаточно для демо, легко заменить на JWT |
| Идемпотентность merge | Повторный вызов не ошибка — возвращает текущее состояние PR |
| PR без ревьюверов | Graceful degradation — PR создаётся, логируется warning |
| Транзакции | `TxManager.Do` / `DoRead` — единая точка управления, изоляция в сервисном слое |
| Ошибки | 3 уровня: storage → domain (с HTTP-кодом) → handler mapping |
| Миграции | golang-migrate, применяются автоматически при старте приложения |
| Docker | Multi-stage build: `golang:1.24-alpine` → `alpine:3.19` (минимальный образ) |
| Логи | zerolog JSON → stdout + файл, индексация через Loki в Grafana |
