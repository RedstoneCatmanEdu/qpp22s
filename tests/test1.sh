#!/bin/bash
# tests/test1.sh

cd "$(dirname "$0")/.."
PROJECT_ROOT=$(pwd)

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}=== TEST 1: Basic QUIC Connection via Load Balancer ===${NC}"

echo "Cleaning up old Mininet instances..."
sudo mn -c > /dev/null 2>&1

echo "Starting Controller..."
python3 controler/run_controler.py > /tmp/test1_controller.log 2>&1 &
CONTROLLER_PID=$!
sleep 2

echo "Starting Mininet and running microclient..."
sudo python3 topo.py <<EOF > /tmp/test1_mininet.log 2>&1
server1 ${PROJECT_ROOT}/bin/server -id 1 &
server2 ${PROJECT_ROOT}/bin/server -id 2 &
client ${PROJECT_ROOT}/bin/microclient -addr 192.168.10.100 -port 443
exit
EOF

kill $CONTROLLER_PID 2>/dev/null

echo -e "\n${GREEN}=== Checking Results ===${NC}"

if grep -q "Good morning!)" /tmp/test1_mininet.log; then
    echo -e "${GREEN}[PASS] Test 1: Microclient successfully connected through the load balancer.${NC}"
else
    echo -e "${RED}[FAIL] Test 1: Microclient failed to get a response.${NC}"
    echo "Look at the logs:"
    tail -n 20 /tmp/test1_mininet.log
fi