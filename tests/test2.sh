#!/bin/bash

cd "$(dirname "$0")/.."
PROJECT_ROOT=$(pwd)

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}=== TEST 2: Round Robin Load Balancing & OVS Traffic ===${NC}"

echo "Cleaning up old Mininet instances..."
sudo mn -c > /dev/null 2>&1

echo "Starting Controller..."
python3 controler/run_controler.py > /tmp/test2_controller.log 2>&1 &
CONTROLLER_PID=$!
sleep 2

echo "Starting Mininet and running 20 clients simultaneously..."

sudo python3 topo.py <<EOF > /tmp/test2_mininet.log 2>&1
server1 ${PROJECT_ROOT}/bin/server -id 1 &
server2 ${PROJECT_ROOT}/bin/server -id 2 &
client ${PROJECT_ROOT}/bin/client -addr 192.168.10.100 -port 443 -c 20
sh ovs-ofctl -O OpenFlow13 dump-flows s1
exit
EOF

kill $CONTROLLER_PID 2>/dev/null

echo -e "\n${GREEN}=== Checking Results (Test 2: Round Robin) ===${NC}"

COUNT_S1=$(grep -c "Round Robin chose Server 1" /tmp/test2_controller.log)
COUNT_S2=$(grep -c "Round Robin chose Server 2" /tmp/test2_controller.log)
COUNT_BYE=$(grep -c "Bye" /tmp/test2_mininet.log)

echo "Server 1 was chosen: $COUNT_S1 times"
echo "Server 2 was chosen: $COUNT_S2 times"
echo "Goodbyes received:  $COUNT_BYE times"

if [ "$COUNT_S1" -ge 9 ] && [ "$COUNT_S1" -le 11 ] && [ "$COUNT_S2" -ge 9 ] && [ "$COUNT_S2" -le 11 ]; then
    echo -e "${GREEN}[PASS] Round Robin distributed clients almost perfectly (approx 10/10).${NC}"
else
    echo -e "${RED}[FAIL] Round Robin distribution is uneven.${NC}"
fi

if [ "$COUNT_BYE" -ge 20 ]; then
    echo -e "${GREEN}[PASS] All 20 clients successfully completed their dialogues.${NC}"
else
    echo -e "${RED}[FAIL] Some clients failed to complete the dialogue (less than 20 Bye messages).${NC}"
fi

echo -e "\n${GREEN}=== Checking Results (Test 7: OVS Direct Traffic) ===${NC}"

OVS_TOTAL_PACKETS=$(grep -oP "(?<=n_packets=)\d+" /tmp/test2_mininet.log | awk '{s+=$1} END {print s}')

CONTROLLER_PACKET_IN=$(grep -c "Packet-In:" /tmp/test2_controller.log)

SERVER_PONGS=$(grep -c "pong" /tmp/test2_mininet.log)

echo "Total packets processed by OVS: $OVS_TOTAL_PACKETS"
echo "Packet-In messages sent to Controller: $CONTROLLER_PACKET_IN"
echo "Total 'pong' responses from servers: $SERVER_PONGS"

if [ "$OVS_TOTAL_PACKETS" -gt 100 ]; then
    echo -e "${GREEN}[PASS] OVS processed a large number of packets directly.${NC}"
else
    echo -e "${RED}[FAIL] OVS did not process enough packets. Maybe rules were not installed?${NC}"
fi

if [ "$CONTROLLER_PACKET_IN" -lt "$SERVER_PONGS" ]; then
    echo -e "${GREEN}[PASS] Controller did NOT process every packet (Packet-In count < actual traffic).${NC}"
else
    echo -e "${RED}[FAIL] Controller seems to be processing every packet. Check table-miss rule.${NC}"
fi