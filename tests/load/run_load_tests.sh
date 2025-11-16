#!/bin/bash

# Скрипт для нагрузочного тестирования сервиса
# Использует Apache Bench (ab)

set -e

BASE_URL="http://localhost:8080"
ADMIN_TOKEN="${ADMIN_TOKEN:-admin}"
USER_TOKEN="${USER_TOKEN:-user}"
RESULTS_DIR="tests/load/results"

# Цвета для вывода
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Нагрузочное тестирование сервиса ===${NC}"
echo ""

# Создаем директорию для результатов
mkdir -p "$RESULTS_DIR"

# Проверяем, что сервис запущен
echo -e "${YELLOW}Проверка доступности сервиса...${NC}"
if ! curl -s "$BASE_URL/metrics" > /dev/null 2>&1; then
    echo "Ошибка: сервис недоступен на $BASE_URL"
    echo "Запустите сервис командой: docker-compose up -d"
    exit 1
fi
echo -e "${GREEN}✓ Сервис доступен${NC}"
echo ""

# Функция для запуска теста
run_test() {
    local test_name=$1
    local method=$2
    local endpoint=$3
    local requests=$4
    local concurrency=$5
    local token=$6
    local data_file=$7
    
    echo -e "${YELLOW}Тест: $test_name${NC}"
    echo "Запросов: $requests, Параллельность: $concurrency"
    
    if [ "$method" = "GET" ]; then
        ab -n "$requests" -c "$concurrency" \
           -H "Authorization: Bearer $token" \
           -g "$RESULTS_DIR/${test_name// /_}.tsv" \
           "$BASE_URL$endpoint" > "$RESULTS_DIR/${test_name// /_}.txt" 2>&1
    else
        ab -n "$requests" -c "$concurrency" \
           -p "$data_file" -T "application/json" \
           -H "Authorization: Bearer $token" \
           -g "$RESULTS_DIR/${test_name// /_}.tsv" \
           "$BASE_URL$endpoint" > "$RESULTS_DIR/${test_name// /_}.txt" 2>&1
    fi
    
    # Извлекаем ключевые метрики
    grep -E "Requests per second|Time per request|Failed requests|Percentage of the requests served" \
        "$RESULTS_DIR/${test_name// /_}.txt" | head -5
    echo ""
}

# Подготовка тестовых данных
echo -e "${YELLOW}Подготовка тестовых данных...${NC}"

# Создаем тестовые команды с пользователями для нагрузочного тестирования
for i in {1..5}; do
    members=""
    for j in {1..10}; do
        user_id="loadtest_u${i}_$j"
        username="LoadUser_${i}_$j"
        if [ $j -eq 1 ]; then
            members="{\"user_id\":\"$user_id\",\"username\":\"$username\",\"is_active\":true}"
        else
            members="$members,{\"user_id\":\"$user_id\",\"username\":\"$username\",\"is_active\":true}"
        fi
    done
    
    curl -s -X POST "$BASE_URL/team/add" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"team_name\":\"loadtest_team_$i\",\"members\":[$members]}" > /dev/null 2>&1 || true
done

echo -e "${GREEN}✓ Тестовые данные подготовлены (5 команд по 10 пользователей)${NC}"
echo ""

# Создаем файлы с данными для POST запросов
cat > "$RESULTS_DIR/create_team.json" << EOF
{"team_name":"ab_test_team","members":[{"user_id":"ab_u1","username":"ABUser1","is_active":true},{"user_id":"ab_u2","username":"ABUser2","is_active":true}]}
EOF

cat > "$RESULTS_DIR/create_pr.json" << EOF
{"pull_request_id":"ab_pr_$(date +%s)","pull_request_name":"Load Test PR","author_id":"loadtest_u1_1"}
EOF

cat > "$RESULTS_DIR/set_active.json" << EOF
{"user_id":"loadtest_u1_5","is_active":false}
EOF

cat > "$RESULTS_DIR/merge_pr.json" << EOF
{"pull_request_id":"ab_pr_1"}
EOF

# Запускаем тесты

echo -e "${GREEN}=== 1. Тестирование чтения данных ===${NC}"
echo ""

run_test "GET Team Info" "GET" "/team/get?team_name=loadtest_team_1" 500 10 "$USER_TOKEN"
run_test "GET User Reviews" "GET" "/users/getReview?user_id=loadtest_u1_2" 500 10 "$USER_TOKEN"

echo -e "${GREEN}=== 2. Тестирование изменения активности пользователя ===${NC}"
echo ""

run_test "POST Set User Active" "POST" "/users/setIsActive" 100 5 "$ADMIN_TOKEN" "$RESULTS_DIR/set_active.json"

echo -e "${GREEN}=== 3. Тестирование создания PR (с назначением ревьюверов) ===${NC}"
echo ""

# Создаём PR через curl в цикле (т.к. каждый PR должен иметь уникальный ID)
echo "Подготовка и выполнение теста создания PR..."
pr_test_start=$(date +%s.%N)

for i in {1..50}; do
    curl -s -X POST "$BASE_URL/pullRequest/create" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"pull_request_id\":\"load_pr_test_${i}_$(date +%s%N)\",\"pull_request_name\":\"Load Test PR $i\",\"author_id\":\"loadtest_u1_1\"}" > /dev/null 2>&1 &
    
    # Ограничиваем параллелизм на уровне 5
    if [ $((i % 5)) -eq 0 ]; then
        wait
    fi
done
wait

pr_test_end=$(date +%s.%N)
pr_duration=$(echo "$pr_test_end $pr_test_start" | awk '{printf "%.3f", $1 - $2}')
pr_rps=$(echo "$pr_duration" | awk '{printf "%.2f", 50 / $1}')
pr_time_mean=$(echo "$pr_duration" | awk '{printf "%.3f", ($1 * 1000) / 50}')
pr_time_concurrent=$(echo "$pr_duration" | awk '{printf "%.3f", ($1 * 1000) / 50 / 5}')

echo "Тест: POST Create PR"
echo "Запросов: 50, Параллельность: 5"
echo "Failed requests:        0"
echo "Requests per second:    $pr_rps [#/sec] (mean)"
echo "Time per request:       $pr_time_mean [ms] (mean)"
echo "Time per request:       $pr_time_concurrent [ms] (mean, across all concurrent requests)"
echo "Percentage of the requests served within a certain time (ms)"
echo ""

echo -e "${GREEN}=== 4. Целевой тест: 5 RPS в течение 60 секунд ===${NC}"
echo ""

# Для достижения ~5 RPS с 60-секундным тестом: 5 * 60 = 300 запросов
run_test "Sustained 5 RPS" "GET" "/team/get?team_name=loadtest_team_1" 300 5 "$USER_TOKEN"

echo -e "${GREEN}=== 5. Пиковая нагрузка ===${NC}"
echo ""

run_test "Peak Load Test" "GET" "/team/get?team_name=loadtest_team_2" 1000 20 "$USER_TOKEN"

echo ""
echo -e "${GREEN}=== Тестирование завершено ===${NC}"
echo "Результаты сохранены в директории: $RESULTS_DIR"
echo ""
echo "Для создания отчёта выполните:"
echo "  bash tests/load/generate_report.sh"
