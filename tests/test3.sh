#!/bin/bash

cd "$(dirname "$0")/.."
PROJECT_ROOT=$(pwd)

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}=== TEST 3: NAT Rebinding & CID Change ===${NC}"

echo "Cleaning up old Mininet instances..."
sudo mn -c > /dev/null 2>&1

echo "Starting Controller..."
python3 controler/run_controler.py > /tmp/test3_controller.log 2>&1 &
CONTROLLER_PID=$!
sleep 2

echo "Starting Mininet and running client with -ChangePort..."
sudo python3 topo.py <<EOF > /tmp/test3_mininet.log 2>&1
server1 ${PROJECT_ROOT}/bin/server -id 1 &
server2 ${PROJECT_ROOT}/bin/server -id 2 &
client ${PROJECT_ROOT}/bin/client -addr 192.168.10.100 -port 443 -ChangePort
exit
EOF

kill $CONTROLLER_PID 2>/dev/null

echo -e "\n${GREEN}=== Checking Results ===${NC}"

REBIND_COUNT=$(grep -c "NAT REBINDING" /tmp/test3_mininet.log)
if [ "$REBIND_COUNT" -ge 1 ]; then
    echo -e "${GREEN}[PASS] NAT Rebinding occurred (client changed its port).${NC}"
else
    echo -e "${RED}[FAIL] NAT Rebinding did not happen.${NC}"
fi

STICKY_COUNT=$(grep -c "Routed by Server CID" /tmp/test3_controller.log)
if [ "$STICKY_COUNT" -ge 1 ]; then
    echo -e "${GREEN}[PASS] Load balancer preserved the connection stickiness (Routed by Server CID).${NC}"
else
    echo -e "${RED}[FAIL] Load balancer applied Round Robin again instead of reusing the server.${NC}"
fi

PORT_BEFORE=$(grep -oP "(?<=you are from 192\.168\.10\.10:)\d+" /tmp/test3_mininet.log | head -n 1)
PORT_AFTER=$(grep -oP "(?<=you are from 192\.168\.10\.10:)\d+" /tmp/test3_mininet.log | tail -n 1)

echo -e "\nClient port BEFORE rebinding: ${PORT_BEFORE}"
echo -e "Client port AFTER rebinding:  ${PORT_AFTER}"

if [ "$PORT_BEFORE" != "$PORT_AFTER" ] && [ -n "$PORT_AFTER" ]; then
    echo -e "${GREEN}[PASS] Server successfully communicated with the client using the NEW port.${NC}"
else
    echo -e "${RED}[FAIL] Port did not change, or dialogue failed.${NC}"
fi

echo -e "\n${GREEN}=== Extracting CID info from Controller Logs ===${NC}"
echo "Original CID parsing (first packet):"
grep "Parsed DCID" /tmp/test3_controller.log | head -n 1

echo "New packet after rebinding (controller sees the same CID):"
grep "Parsed DCID" /tmp/test3_controller.log | tail -n 1