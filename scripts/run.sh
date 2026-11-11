#!/bin/bash
cd "$(dirname "$0")/.."
export PROJECT_ROOT=$(pwd)
mkdir -p logs

echo "=== Starting Controller in background ==="
echo "Logs: logs/controler.log"
python3 controler/run_controler.py > logs/controler.log 2>&1 &
CONTROLLER_PID=$!
sleep 2
W
echo "=== Starting Mininet & Servers ==="
echo "Servers are started automatically in the background."
echo "Type 'exit' in Mininet to stop the topology and the controller."

sudo -E python3 topo.py -a

echo "=== Mininet stopped. Killing Controller... ==="
kill $CONTROLLER_PID 2>/dev/null || true
echo "=== Done! ==="