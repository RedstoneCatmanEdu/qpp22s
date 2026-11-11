#!/bin/bash
cd "$(dirname "$0")/.."

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Запрашиваем пароль sudo один раз для всех тестов
sudo -v

echo -e "${GREEN}=== RUNNING ALL SYSTEM TESTS ===${NC}\n"

TOTAL_TESTS=4
PASSED_TESTS=0

run_test() {
    local test_name=$1
    local test_script=$2
    
    echo ">>> Running $test_name..."
    
    local output
    output=$($test_script 2>&1)
    
    if echo "$output" | grep -q "\[FAIL\]"; then
        echo -e "${RED}    [FAILED] $test_name${NC}"
    else
        echo -e "${GREEN}    [PASSED] $test_name${NC}"
        ((PASSED_TESTS++))
    fi
}

run_test "Test 1 (Basic Connection)" "./tests/test1.sh"
run_test "Test 2 (Round Robin & OVS)" "./tests/test2.sh"
run_test "Test 3 (NAT Rebinding & CID)" "./tests/test3.sh"
run_test "Test 4 (Malformed Packets)" "./tests/test4.sh"

echo -e "\n========================================="
if [ "$PASSED_TESTS" -eq "$TOTAL_TESTS" ]; then
    echo -e "${GREEN}=== SUCCESS: ALL $TOTAL_TESTS TESTS PASSED! ===${NC}"
else
    echo -e "${RED}=== WARNING: ONLY $PASSED_TESTS OUT OF $TOTAL_TESTS TESTS PASSED ===${NC}"
fi
echo -e "========================================="