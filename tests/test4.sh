#!/bin/bash

cd "$(dirname "$0")/.."
PROJECT_ROOT=$(pwd)

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}=== TEST 6: Handling Malformed Packets ===${NC}"

echo "Cleaning up old Mininet instances and controllers..."
sudo mn -c > /dev/null 2>&1
pkill -f run_controler.py || true
sleep 1

echo "Starting Controller..."
python3 controler/run_controler.py > /tmp/test4_controller.log 2>&1 &
CONTROLLER_PID=$!
sleep 3 # Даем контроллеру больше времени на подключение

echo "Starting Mininet and sending garbage packets..."
sudo python3 topo.py <<EOF > /tmp/test4_mininet.log 2>&1
server1 ${PROJECT_ROOT}/bin/server -id 1 &
server2 ${PROJECT_ROOT}/bin/server -id 2 &
sleep 2
client python3 ${PROJECT_ROOT}/tests/bad_client.py
sleep 3
exit
EOF

kill $CONTROLLER_PID 2>/dev/null

echo -e "\n${GREEN}=== Checking Results ===${NC}"

if grep -q "ERROR:" /tmp/test4_controller.log; then
    echo -e "${RED}[FAIL] Controller crashed while processing malformed packets!${NC}"
    grep -A 5 "ERROR:" /tmp/test4_controller.log
else
    echo -e "${GREEN}[PASS] Controller did NOT crash while processing packets.${NC}"
fi

if grep -q "Packet-In:" /tmp/test4_controller.log; then
    echo -e "${GREEN}[PASS] Controller received the packets (Packet-In logged).${NC}"
else
    echo -e "${RED}[FAIL] Controller did not even receive the packets!${NC}"
    echo "Look at Mininet log:"
    tail -n 10 /tmp/test4_mininet.log
    echo "Look at Controller log:"
    cat /tmp/test4_controller.log
fi

if grep -q "Not a valid QUIC packet" /tmp/test4_controller.log; then
    echo -e "${GREEN}[PASS] Controller gracefully handled the garbage packets.${NC}"
else
    echo -e "${RED}[FAIL] Controller did not log 'Not a valid QUIC packet'.${NC}"
    echo -e "\n--- Controller Log ---"
    cat /tmp/test4_controller.log
fi