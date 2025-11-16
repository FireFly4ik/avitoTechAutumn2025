.PHONY: help build run clean

# Переменные
BINARY_NAME=main
DOCKER_COMPOSE=docker-compose
DOCKER_COMPOSE_TEST=docker-compose -f docker-compose.test.yml

# Цвета для вывода
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

##@ Основные команды

help: ## Показать список доступных команд
	@echo "Доступные команды:"
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

build: ## Собрать бинарный файл
	@echo "Сборка приложения..."
	go build -o $(BINARY_NAME) ./cmd/api

run: ## Запустить приложение локально
	go run ./cmd/api/main.go

clean: ## Очистить сгенерированные файлы
	@echo "Очистка..."
	rm -f $(BINARY_NAME)
	rm -rf tests/load/results/*
	go clean

##@ Docker команды

docker-up: ## Запустить production окружение (БД + API + мониторинг)
	@echo "$(YELLOW)Запуск production окружения...$(NC)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)✅ Production окружение запущено$(NC)"
	@echo "API: http://localhost:8080"
	@echo "Grafana: http://localhost:3000"
	@echo "Prometheus: http://localhost:9090"

docker-down: ## Остановить production окружение
	@echo "Остановка production окружения..."
	$(DOCKER_COMPOSE) down

docker-restart: ## Перезапустить production окружение
	@echo "Перезапуск production окружения..."
	$(DOCKER_COMPOSE) restart

docker-logs: ## Показать логи production API
	$(DOCKER_COMPOSE) logs -f api

docker-test-up: ## Запустить тестовое окружение (тестовая БД + API)
	@echo "$(YELLOW)Запуск тестового окружения...$(NC)"
	$(DOCKER_COMPOSE_TEST) up -d --build
	@echo "Ожидание готовности сервисов..."
	@sleep 5
	@echo "$(GREEN)✅ Тестовое окружение запущено$(NC)"
	@echo "Test API: http://localhost:8081"
	@echo "Test DB: localhost:5436"

docker-test-down: ## Остановить тестовое окружение
	@echo "Остановка тестового окружения..."
	$(DOCKER_COMPOSE_TEST) down

docker-test-logs: ## Показать логи тестового API
	$(DOCKER_COMPOSE_TEST) logs -f test_api

docker-clean: ## Удалить все контейнеры и volumes
	@echo "Очистка всех Docker ресурсов..."
	$(DOCKER_COMPOSE) down -v
	$(DOCKER_COMPOSE_TEST) down -v
	docker system prune -f

##@ Тестирование

test-unit: ## Запустить unit тесты
	@echo "$(YELLOW)Запуск unit тестов...$(NC)"
	go test -v ./tests/unit/...

test-unit-cover: ## Запустить unit тесты с покрытием
	@echo "$(YELLOW)Запуск unit тестов с покрытием...$(NC)"
	go test -v -coverprofile=coverage.out -covermode=atomic ./tests/unit/... -coverpkg=./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total | awk '{print "Общее покрытие: " $$3}'
	@echo "$(GREEN)✅ Отчёт о покрытии: coverage.html$(NC)"

test-integration: docker-test-up ## Запустить интеграционные тесты (автоматически поднимает тестовую среду)
	@echo "$(YELLOW)Запуск интеграционных тестов...$(NC)"
	@export TEST_DB_HOST=localhost TEST_DB_PORT=5436 TEST_DB_NAME=avito_test TEST_DB_USER=avito TEST_DB_PASSWORD=avito_password; \
	go test -v ./tests/integration/... || ($(DOCKER_COMPOSE_TEST) down && exit 1)
	@$(DOCKER_COMPOSE_TEST) down
	@echo "$(GREEN)✅ Интеграционные тесты завершены$(NC)"

test-e2e: docker-test-up ## Запустить E2E тесты (автоматически поднимает тестовую среду)
	@echo "$(YELLOW)Запуск E2E тестов...$(NC)"
	@export TEST_API_URL=http://localhost:8081 ADMIN_TOKEN=admin USER_TOKEN=user; \
	go test -v ./tests/e2e/... || ($(DOCKER_COMPOSE_TEST) down && exit 1)
	@$(DOCKER_COMPOSE_TEST) down
	@echo "$(GREEN)✅ E2E тесты завершены$(NC)"

test-load: ## Запустить нагрузочные тесты (требует запущенный API)
	@echo "$(YELLOW)Запуск нагрузочных тестов...$(NC)"
	@if ! curl -s http://localhost:8080/metrics > /dev/null 2>&1; then \
		echo "$(YELLOW)API не запущен, поднимаю production окружение...$(NC)"; \
		$(DOCKER_COMPOSE) up -d; \
		sleep 5; \
	fi
	bash tests/load/run_load_tests.sh
	bash tests/load/generate_report.sh
	@echo "$(GREEN)✅ Отчёт по нагрузке: tests/load/LOAD_TESTING.md$(NC)"

test-all: ## Запустить ВСЕ тесты последовательно (unit -> integration -> e2e -> load)
	@echo "$(YELLOW)========================================$(NC)"
	@echo "$(YELLOW)  ЗАПУСК ВСЕХ ТЕСТОВ$(NC)"
	@echo "$(YELLOW)========================================$(NC)"
	bash tests/run_all_tests.sh

test-quick: test-unit ## Быстрые тесты (только unit)
	@echo "$(GREEN)✅ Быстрые тесты завершены$(NC)"

##@ Качество кода

lint: ## Запустить линтер (golangci-lint)
	@echo "$(YELLOW)Запуск линтера...$(NC)"
	golangci-lint run
	@echo "$(GREEN)✅ Линтер завершён$(NC)"

lint-fix: ## Запустить линтер с автоисправлением
	@echo "$(YELLOW)Запуск линтера с автоисправлением...$(NC)"
	golangci-lint run --fix
	@echo "$(GREEN)✅ Код исправлен$(NC)"

fmt: ## Форматировать код
	@echo "Форматирование кода..."
	go fmt ./...
	@echo "$(GREEN)✅ Код отформатирован$(NC)"

vet: ## Проверить код (go vet)
	@echo "Проверка кода..."
	go vet ./...
	@echo "$(GREEN)✅ Проверка завершена$(NC)"

mocks: ## Сгенерировать все моки (mockery)
	@echo "$(YELLOW)Генерация моков...$(NC)"
	go generate ./internal/domain/...
	go generate ./internal/storage/...
	@echo "$(GREEN)✅ Моки сгенерированы в internal/mocks/$(NC)"

##@ Зависимости

deps: ## Установить зависимости
	@echo "Установка зависимостей..."
	go mod download
	go mod tidy
	@echo "$(GREEN)✅ Зависимости установлены$(NC)"

deps-update: ## Обновить зависимости
	@echo "Обновление зависимостей..."
	go get -u ./...
	go mod tidy
	@echo "$(GREEN)✅ Зависимости обновлены$(NC)"

##@ CI/CD команды

ci-test: lint test-unit ## CI pipeline: линтер + unit тесты
	@echo "$(GREEN)✅ CI тесты пройдены$(NC)"

ci-full: lint test-all ## CI pipeline: линтер + все тесты
	@echo "$(GREEN)✅ Полный CI pipeline завершён$(NC)"

##@ Разработка

dev: docker-up ## Запустить development окружение
	@echo "$(GREEN)✅ Development окружение готово$(NC)"

dev-logs: ## Следить за логами в режиме разработки
	$(DOCKER_COMPOSE) logs -f

check: lint test-unit ## Быстрая проверка перед коммитом (линтер + unit тесты)
	@echo "$(GREEN)✅ Проверка завершена успешно$(NC)"

pre-commit: fmt lint test-unit ## Полная проверка перед коммитом
	@echo "$(GREEN)✅ Готово к коммиту$(NC)"

##@ Информация

status: ## Показать статус всех сервисов
	@echo "=== Production окружение ==="
	@$(DOCKER_COMPOSE) ps
	@echo ""
	@echo "=== Тестовое окружение ==="
	@$(DOCKER_COMPOSE_TEST) ps

ports: ## Показать используемые порты
	@echo "=== Используемые порты ==="
	@echo "Production:"
	@echo "  API:        8080"
	@echo "  PostgreSQL: 5433"
	@echo "  Prometheus: 9090"
	@echo "  Grafana:    3000"
	@echo ""
	@echo "Testing:"
	@echo "  API:        8081"
	@echo "  PostgreSQL: 5436"

info: ## Показать информацию о проекте
	@echo "=== Информация о проекте ==="
	@echo "Название: PR Reviewer Assignment Service"
	@echo "Язык: Go $(shell go version | awk '{print $$3}')"
	@echo "Docker Compose: $(shell docker-compose version --short)"
	@echo ""
	@echo "Структура тестов:"
	@echo "  Unit:        $(shell find tests/unit -name '*_test.go' | wc -l | tr -d ' ') файлов"
	@echo "  Integration: $(shell find tests/integration -name '*_test.go' | wc -l | tr -d ' ') файлов"
	@echo "  E2E:         $(shell find tests/e2e -name '*_test.go' | wc -l | tr -d ' ') файлов"
	@echo ""
	@echo "Моки: $(shell find internal/mocks -name '*.go' 2>/dev/null | wc -l | tr -d ' ') файлов"
