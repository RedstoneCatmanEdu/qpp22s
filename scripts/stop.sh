#!/bin/bash
cd "$(dirname "$0")/.."

echo "=== Stopping all processes and cleaning Mininet ==="
sudo mn -c > /dev/null 2>&1
pkill -f run_controler.py || true
pkill -f bin/server || true
pkill -f bin/client || true

echo "=== Cleaning up logs ==="
# Удаляем логи от run.sh
rm -rf logs/*.log 2>/dev/null || true
# Удаляем логи от тестов
rm -f /tmp/test1_*.log /tmp/test2_*.log /tmp/test3_*.log /tmp/test4_*.log 2>/dev/null || true
rm -f /tmp/server1.log /tmp/server2.log 2>/dev/null || true

echo "=== Cleanup complete! ==="