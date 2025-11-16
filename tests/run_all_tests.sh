#!/bin/bash

# Скрипт для последовательного запуска всех типов тестов
# Unit -> Integration -> E2E -> Load Testing

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция для вывода заголовков
print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
}

# Функция для вывода успеха
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Функция для вывода ошибки
print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Функция для вывода предупреждения
print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Переменные для статистики
UNIT_STATUS="❌"
INTEGRATION_STATUS="❌"
E2E_STATUS="❌"
LOAD_STATUS="❌"

START_TIME=$(date +%s)

print_header "ЗАПУСК ПОЛНОГО НАБОРА ТЕСТОВ"

# ============================================
# 1. UNIT ТЕСТЫ
# ============================================
print_header "1/4 - Unit тесты"
echo "Запуск unit тестов (не требуют БД)..."

if go test -v ./tests/unit/... 2>&1 | tee /tmp/unit_tests.log; then
    print_success "Unit тесты пройдены"
    UNIT_STATUS="✅"
else
    print_error "Unit тесты провалились"
    echo "Смотрите лог: /tmp/unit_tests.log"
    # Не прерываем выполнение, продолжаем с остальными тестами
fi

# ============================================
# 2. ИНТЕГРАЦИОННЫЕ ТЕСТЫ
# ============================================
print_header "2/4 - Интеграционные тесты"

echo "Поднимаем тестовую БД и API..."
docker-compose -f docker-compose.test.yml up -d --build

echo "Ожидаем готовности сервисов..."
sleep 5

# Проверяем, что сервисы запустились
if ! docker-compose -f docker-compose.test.yml ps | grep -q "Up"; then
    print_error "Не удалось запустить тестовые сервисы"
    docker-compose -f docker-compose.test.yml logs
    docker-compose -f docker-compose.test.yml down
    exit 1
fi

print_success "Тестовые сервисы запущены"

# Настраиваем переменные окружения для интеграционных тестов
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5436
export TEST_DB_NAME=avito_test
export TEST_DB_USER=avito
export TEST_DB_PASSWORD=avito_password

echo "Запуск интеграционных тестов..."
if go test -v ./tests/integration/... 2>&1 | tee /tmp/integration_tests.log; then
    print_success "Интеграционные тесты пройдены"
    INTEGRATION_STATUS="✅"
else
    print_error "Интеграционные тесты провалились"
    echo "Смотрите лог: /tmp/integration_tests.log"
fi

# ============================================
# 3. E2E ТЕСТЫ
# ============================================
print_header "3/4 - E2E тесты"

echo "Запуск E2E тестов (используем уже запущенные сервисы)..."
export TEST_API_URL=http://localhost:8081
export ADMIN_TOKEN=admin
export USER_TOKEN=user

if go test -v ./tests/e2e/... 2>&1 | tee /tmp/e2e_tests.log; then
    print_success "E2E тесты пройдены"
    E2E_STATUS="✅"
else
    print_error "E2E тесты провалились"
    echo "Смотрите лог: /tmp/e2e_tests.log"
fi

# ============================================
# 4. НАГРУЗОЧНЫЕ ТЕСТЫ
# ============================================
print_header "4/4 - Нагрузочные тесты"

echo "Запуск нагрузочных тестов..."

# Меняем BASE_URL в скрипте на тестовый порт
export TEST_BASE_URL=http://localhost:8081

# Создаем временную копию скрипта с правильным URL
sed 's|BASE_URL="http://localhost:8080"|BASE_URL="http://localhost:8081"|g' tests/load/run_load_tests.sh > /tmp/run_load_tests_tmp.sh
chmod +x /tmp/run_load_tests_tmp.sh

if bash /tmp/run_load_tests_tmp.sh 2>&1 | tee /tmp/load_tests.log; then
    print_success "Нагрузочные тесты пройдены"
    
    # Генерируем отчёт
    echo "Генерация отчёта..."
    bash tests/load/generate_report.sh
    LOAD_STATUS="✅"
else
    print_error "Нагрузочные тесты провалились"
    echo "Смотрите лог: /tmp/load_tests.log"
fi

# Очистка
rm -f /tmp/run_load_tests_tmp.sh

# ============================================
# ЗАВЕРШЕНИЕ
# ============================================
print_header "Остановка тестовых сервисов"
docker-compose -f docker-compose.test.yml down

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

print_header "ИТОГОВЫЙ ОТЧЁТ"

echo ""
echo "┌─────────────────────────────────────────┐"
echo "│         Результаты тестирования         │"
echo "├─────────────────────────────────────────┤"
echo "│ ${UNIT_STATUS}  Unit тесты                         │"
echo "│ ${INTEGRATION_STATUS}  Интеграционные тесты             │"
echo "│ ${E2E_STATUS}  E2E тесты                         │"
echo "│ ${LOAD_STATUS}  Нагрузочные тесты                │"
echo "├─────────────────────────────────────────┤"
echo "│ Общее время: ${DURATION}s                     │"
echo "└─────────────────────────────────────────┘"
echo ""

# Проверяем, все ли тесты прошли
if [ "$UNIT_STATUS" = "✅" ] && [ "$INTEGRATION_STATUS" = "✅" ] && [ "$E2E_STATUS" = "✅" ] && [ "$LOAD_STATUS" = "✅" ]; then
    print_success "ВСЕ ТЕСТЫ ПРОЙДЕНЫ!"
    echo ""
    echo "Отчёты:"
    echo "  - Unit тесты: /tmp/unit_tests.log"
    echo "  - Интеграционные: /tmp/integration_tests.log"
    echo "  - E2E: /tmp/e2e_tests.log"
    echo "  - Нагрузочные: /tmp/load_tests.log"
    echo "  - Отчёт по нагрузке: LOAD_TESTING.md"
    exit 0
else
    print_error "НЕКОТОРЫЕ ТЕСТЫ НЕ ПРОШЛИ"
    echo ""
    echo "Проверьте логи:"
    [ "$UNIT_STATUS" = "❌" ] && echo "  - Unit: /tmp/unit_tests.log"
    [ "$INTEGRATION_STATUS" = "❌" ] && echo "  - Integration: /tmp/integration_tests.log"
    [ "$E2E_STATUS" = "❌" ] && echo "  - E2E: /tmp/e2e_tests.log"
    [ "$LOAD_STATUS" = "❌" ] && echo "  - Load: /tmp/load_tests.log"
    exit 1
fi
